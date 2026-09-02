package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mythezone/navidrome-music-room/gateway/internal/domain"
)

var (
	// Navidrome's native web login intentionally generates a three-byte salt,
	// encoded as six hexadecimal characters (server/auth.go).
	validSalt  = regexp.MustCompile(`^[A-Fa-f0-9]{6,64}$`)
	validToken = regexp.MustCompile(`^[A-Fa-f0-9]{32}$`)
)

type NavidromeClient struct {
	baseURL *url.URL
	client  *http.Client
}

func NewNavidromeClient(baseURL *url.URL) *NavidromeClient {
	return &NavidromeClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 8 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

type subsonicEnvelope struct {
	Response subsonicResponse `json:"subsonic-response"`
}

type subsonicResponse struct {
	Status string         `json:"status"`
	User   *subsonicUser  `json:"user,omitempty"`
	Song   *subsonicSong  `json:"song,omitempty"`
	Error  *subsonicError `json:"error,omitempty"`
}

type subsonicError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type subsonicUser struct {
	Username  string `json:"username"`
	AdminRole bool   `json:"adminRole"`
	Folder    []int  `json:"folder"`
}

type subsonicSong struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	Artist   string  `json:"artist"`
	Album    string  `json:"album"`
	AlbumID  string  `json:"albumId"`
	CoverArt string  `json:"coverArt"`
	Duration float64 `json:"duration"`
}

func (c *NavidromeClient) Verify(ctx context.Context, proof domain.AuthProof) (domain.User, error) {
	if err := validateProof(proof); err != nil {
		return domain.User{}, err
	}
	query := c.authQuery(proof)
	query.Set("username", proof.Username)
	response, err := c.request(ctx, "getUser", query)
	if err != nil {
		return domain.User{}, err
	}
	if response.User == nil || !domain.EqualUsername(response.User.Username, proof.Username) {
		return domain.User{}, domain.NewError(401, "navidrome_auth_failed", "Navidrome did not return the authenticated user")
	}
	folders := response.User.Folder
	if folders == nil {
		folders = make([]int, 0)
	}
	return domain.User{
		Username:       response.User.Username,
		DisplayName:    response.User.Username,
		Admin:          response.User.AdminRole,
		MusicFolderIDs: folders,
	}, nil
}

func (c *NavidromeClient) ValidateTrack(ctx context.Context, proof domain.AuthProof, requested domain.NavidromeTrackRef) (domain.NavidromeTrackRef, error) {
	if strings.TrimSpace(requested.ID) == "" {
		return domain.NavidromeTrackRef{}, domain.NewError(400, "track_invalid", "Track ID is required")
	}
	query := c.authQuery(proof)
	query.Set("id", requested.ID)
	response, err := c.request(ctx, "getSong", query)
	if err != nil {
		return domain.NavidromeTrackRef{}, err
	}
	if response.Song == nil || response.Song.ID != requested.ID {
		return domain.NavidromeTrackRef{}, domain.NewError(404, "track_not_found", "Track is not available to this Navidrome user")
	}
	return domain.NavidromeTrackRef{
		ID:              response.Song.ID,
		MusicFolderID:   requested.MusicFolderID,
		AlbumID:         response.Song.AlbumID,
		Title:           response.Song.Title,
		Artist:          response.Song.Artist,
		Album:           response.Song.Album,
		DurationSeconds: response.Song.Duration,
		CoverArtID:      response.Song.CoverArt,
	}, nil
}

func (c *NavidromeClient) authQuery(proof domain.AuthProof) url.Values {
	query := url.Values{}
	query.Set("u", proof.Username)
	query.Set("t", proof.Token)
	query.Set("s", proof.Salt)
	query.Set("v", "1.16.1")
	query.Set("c", "MusicMate")
	query.Set("f", "json")
	return query
}

func (c *NavidromeClient) request(ctx context.Context, method string, query url.Values) (subsonicResponse, error) {
	endpoint := *c.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/rest/" + method + ".view"
	endpoint.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return subsonicResponse{}, err
	}
	request.Header.Set("Accept", "application/json")
	response, err := c.client.Do(request)
	if err != nil {
		return subsonicResponse{}, domain.NewError(502, "navidrome_unavailable", "Navidrome authentication service is unavailable")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return subsonicResponse{}, domain.ErrorWithDetails(502, "navidrome_bad_response", "Navidrome returned an unexpected HTTP response", map[string]int{"status": response.StatusCode})
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return subsonicResponse{}, err
	}
	var envelope subsonicEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		return subsonicResponse{}, domain.NewError(502, "navidrome_bad_response", "Navidrome returned malformed JSON")
	}
	if envelope.Response.Status != "ok" {
		message := "Navidrome rejected the request"
		code := 0
		if envelope.Response.Error != nil {
			code = envelope.Response.Error.Code
			if strings.TrimSpace(envelope.Response.Error.Message) != "" {
				message = envelope.Response.Error.Message
			}
		}
		status := 401
		if method != "getUser" && (code == 70 || code == 0) {
			status = 404
		}
		return subsonicResponse{}, domain.ErrorWithDetails(status, "navidrome_request_rejected", message, map[string]int{"subsonicCode": code})
	}
	return envelope.Response, nil
}

func validateProof(proof domain.AuthProof) error {
	username := strings.TrimSpace(proof.Username)
	if username == "" || len(username) > 128 {
		return domain.NewError(400, "auth_proof_invalid", "Username is invalid")
	}
	if !validSalt.MatchString(proof.Salt) || !validToken.MatchString(proof.Token) {
		return domain.NewError(400, "auth_proof_invalid", "OpenSubsonic salt/token proof is invalid")
	}
	return nil
}

func folderDetails(required, allowed []int) map[string][]string {
	convert := func(values []int) []string {
		result := make([]string, len(values))
		for index, value := range values {
			result[index] = strconv.Itoa(value)
		}
		return result
	}
	return map[string][]string{"required": convert(required), "allowed": convert(allowed)}
}

func (c *NavidromeClient) String() string {
	return fmt.Sprintf("NavidromeClient(%s)", c.baseURL.Redacted())
}
