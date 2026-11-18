import { readonly, ref } from 'vue'

export const locales = ['en', 'zh-CN', 'zh-TW', 'ja', 'es', 'fr', 'de'] as const
export type Locale = typeof locales[number]
export const localeNames: Record<Locale, string> = { en: 'English', 'zh-CN': '简体中文', 'zh-TW': '繁體中文', ja: '日本語', es: 'Español', fr: 'Français', de: 'Deutsch' }

// Each row contains the key followed by all seven locales, in the order above.
export const messageRows = `
appTitle|Listen together|一起听歌|一起聽歌|一緒に聴こう|Escuchemos juntos|Écoutons ensemble|Gemeinsam hören
language|Language|语言|語言|言語|Idioma|Langue|Sprache
playing|Playing|播放|播放|再生|Reproducción|Lecture|Wiedergabe
queue|Queue|待播放|待播放|再生待ち|Cola|File d’attente|Warteschlange
history|History|历史|歷史|履歴|Historial|Historique|Verlauf
catalog|Music library|点歌台|點歌台|ライブラリ|Biblioteca|Bibliothèque|Musikbibliothek
favorites|Favorites|收藏|收藏|お気に入り|Favoritos|Favoris|Favoriten
chat|Chat|群聊|群聊|チャット|Chat|Discussion|Chat
loading|Loading…|正在加载…|正在載入…|読み込み中…|Cargando…|Chargement…|Wird geladen…
joining|Joining the room…|正在加入房间…|正在加入房間…|ルームに参加中…|Entrando en la sala…|Connexion au salon…|Raum wird betreten…
unavailable|Unable to enter this room|暂时无法进入这个房间|暫時無法進入這個房間|ルームに入れません|No se puede entrar en esta sala|Impossible d’entrer dans ce salon|Dieser Raum ist nicht verfügbar
backNavidrome|Back to Navidrome|返回 Navidrome|返回 Navidrome|Navidrome に戻る|Volver a Navidrome|Retour à Navidrome|Zurück zu Navidrome
retry|Retry|重试|重試|再試行|Reintentar|Réessayer|Erneut versuchen
loginTitle|Join with Navidrome|使用 Navidrome 加入房间|使用 Navidrome 加入房間|Navidrome で参加|Entrar con Navidrome|Rejoindre avec Navidrome|Mit Navidrome beitreten
loginHint|Use your own account and library access.|使用自己的账号和音乐库权限。|使用自己的帳號和音樂庫權限。|自分のアカウントで音楽を楽しめます。|Usa tu cuenta y tus permisos de biblioteca.|Utilisez votre compte et vos accès à la bibliothèque.|Nutze dein Konto und deine Bibliotheksrechte.
username|Username|用户名|使用者名稱|ユーザー名|Usuario|Identifiant|Benutzername
password|Password|密码|密碼|パスワード|Contraseña|Mot de passe|Passwort
showPassword|Show password|显示密码|顯示密碼|パスワードを表示|Mostrar contraseña|Afficher le mot de passe|Passwort anzeigen
hidePassword|Hide password|隐藏密码|隱藏密碼|パスワードを隠す|Ocultar contraseña|Masquer le mot de passe|Passwort verbergen
login|Sign in and join|登录并加入|登入並加入|ログインして参加|Iniciar sesión y entrar|Se connecter et rejoindre|Anmelden und beitreten
connected|Following room playback|跟随房间播放|跟隨房間播放|ルームと同期中|Siguiendo la reproducción|Lecture synchronisée|Mit dem Raum synchronisiert
reconnecting|Reconnecting…|正在重新连接…|正在重新連線…|再接続中…|Reconectando…|Reconnexion…|Verbindung wird wiederhergestellt…
onlineCount|{count} online|{count} 人在线|{count} 人在線|{count} 人オンライン|{count} en línea|{count} en ligne|{count} online
share|Share|分享|分享|共有|Compartir|Partager|Teilen
settings|Room settings|房间设置|房間設定|ルーム設定|Ajustes de la sala|Paramètres du salon|Raumeinstellungen
profile|Member profile|成员档案|成員檔案|メンバープロフィール|Perfil del miembro|Profil du membre|Mitgliedsprofil
update|Update available: {version}|有新版本：{version}|有新版本：{version}|更新があります：{version}|Actualización: {version}|Mise à jour : {version}|Update verfügbar: {version}
close|Close|关闭|關閉|閉じる|Cerrar|Fermer|Schließen
save|Save|保存|儲存|保存|Guardar|Enregistrer|Speichern
saved|Saved|已保存|已儲存|保存しました|Guardado|Enregistré|Gespeichert
cancel|Cancel|取消|取消|キャンセル|Cancelar|Annuler|Abbrechen
remove|Remove|移除|移除|削除|Quitar|Retirer|Entfernen
delete|Delete|删除|刪除|削除|Eliminar|Supprimer|Löschen
confirmRemove|Remove {name} from this room?|确定从房间移除 {name}？|確定從房間移除 {name}？|{name} をルームから削除しますか？|¿Quitar a {name} de la sala?|Retirer {name} de ce salon ?|{name} aus dem Raum entfernen?
confirmDelete|Delete this item?|确定删除此项目？|確定刪除此項目？|削除しますか？|¿Eliminar este elemento?|Supprimer cet élément ?|Diesen Eintrag löschen?
leave|Leave room|离开房间|離開房間|ルームを退出|Salir de la sala|Quitter le salon|Raum verlassen
confirmLeave|Leave this room?|确定离开房间？|確定離開房間？|ルームを退出しますか？|¿Salir de esta sala?|Quitter ce salon ?|Diesen Raum verlassen?
empty|Nothing here yet|这里还没有内容|這裡還沒有內容|まだありません|Todavía no hay nada|Rien pour le moment|Hier ist es noch leer
sync|Sync playback|同步播放|同步播放|再生を同期|Sincronizar reproducción|Synchroniser la lecture|Wiedergabe synchronisieren
synced|Playback synchronized|已同步播放进度|已同步播放進度|同期しました|Reproducción sincronizada|Lecture synchronisée|Wiedergabe synchronisiert
waiting|Waiting for a song|等待点歌|等待點歌|曲を待っています|Esperando una canción|En attente d’un morceau|Warte auf einen Titel
chooseSong|Choose a song from the library|从点歌台选择一首歌曲开始|從點歌台選擇一首歌曲開始|ライブラリから曲を選んでください|Elige una canción de la biblioteca|Choisissez un morceau dans la bibliothèque|Wähle einen Titel aus der Bibliothek
requestedBy|Requested by {name}|由 {name} 点播|由 {name} 點播|{name} のリクエスト|Pedida por {name}|Demandé par {name}|Gewünscht von {name}
listen|Start listening|开始收听|開始收聽|聴き始める|Empezar a escuchar|Commencer à écouter|Zuhören starten
pauseLocal|Pause locally|暂停收听|暫停收聽|自分の再生を一時停止|Pausar localmente|Mettre en pause pour moi|Lokal pausieren
localHint|Only affects your listening|仅影响你的收听|僅影響你的收聽|自分の再生のみ変更|Solo afecta a tu escucha|Concerne uniquement votre écoute|Betrifft nur deine Wiedergabe
playRoom|Play room|播放房间|播放房間|ルームを再生|Reproducir sala|Lancer le salon|Raum starten
pauseRoom|Pause room|暂停房间|暫停房間|ルームを一時停止|Pausar sala|Mettre le salon en pause|Raum pausieren
previous|Previous track|上一首|上一首|前の曲|Pista anterior|Morceau précédent|Vorheriger Titel
next|Next / Skip track|下一首 / 跳过|下一首 / 跳過|次の曲へスキップ|Siguiente / Saltar|Suivant / Passer|Weiter / Überspringen
skipRoom|Everyone in this room can skip|房间允许所有成员跳歌|房間允許所有成員跳歌|全員がスキップできます|Todos pueden saltar canciones|Tous les membres peuvent passer un morceau|Alle dürfen Titel überspringen
skipOwn|You can skip your own requested track|你可以跳过自己点播的歌曲|你可以跳過自己點播的歌曲|自分のリクエストをスキップできます|Puedes saltar tu propia canción|Vous pouvez passer votre propre demande|Du darfst deinen eigenen Musikwunsch überspringen
progress|Playback position|播放进度|播放進度|再生位置|Posición de reproducción|Position de lecture|Wiedergabeposition
volume|Volume|音量|音量|音量|Volumen|Volume|Lautstärke
lyrics|Lyrics|歌词|歌詞|歌詞|Letra|Paroles|Liedtext
lyricsFocus|Expand / collapse lyrics|展开 / 收起歌词|展開 / 收起歌詞|歌詞を拡大・縮小|Ampliar / reducir letra|Agrandir / réduire les paroles|Liedtext vergrößern / verkleinern
immersive|Immersive listening|沉浸聆听|沉浸聆聽|没入モード|Escucha inmersiva|Écoute immersive|Immersives Hören
exitImmersive|Exit immersive|退出沉浸模式|退出沉浸模式|没入モードを終了|Salir del modo inmersivo|Quitter le mode immersif|Immersiven Modus verlassen
resize|Resize player and workspace|调整播放器与工作区宽度|調整播放器與工作區寬度|プレーヤーと作業領域の幅を調整|Ajustar reproductor y panel|Redimensionner le lecteur et le panneau|Player und Arbeitsbereich anpassen
autoplay|Tap Start listening to enable audio.|请点击“开始收听”启用音频。|請點擊「開始收聽」啟用音訊。|「聴き始める」を押して音声を有効にします。|Pulsa Empezar a escuchar para activar el audio.|Appuyez sur Commencer à écouter pour activer l’audio.|Tippe auf Zuhören starten, um Audio zu aktivieren.
eq|Equalizer|均衡器|等化器|イコライザー|Ecualizador|Égaliseur|Equalizer
eqEnable|Enable EQ|启用均衡器|啟用等化器|EQ を有効にする|Activar ecualizador|Activer l’égaliseur|EQ aktivieren
eqMobile|Native audio on mobile preserves background playback. EQ is bypassed.|移动端使用原生音频保持后台播放，均衡器不生效。|行動裝置使用原生音訊保持背景播放，等化器不生效。|モバイルではバックグラウンド再生のため EQ を使用しません。|En móviles se omite el EQ para mantener el audio en segundo plano.|Sur mobile, l’égaliseur est désactivé pour préserver la lecture en arrière-plan.|Auf Mobilgeräten bleibt EQ für die Hintergrundwiedergabe deaktiviert.
flat|Flat|平直|平直|フラット|Plano|Neutre|Neutral
bass|Bass|低音|低音|低音|Graves|Basses|Bass
vocal|Vocal|人声|人聲|ボーカル|Voces|Voix|Gesang
warm|Warm|温暖|溫暖|ウォーム|Cálido|Chaleureux|Warm
bright|Bright|明亮|明亮|ブライト|Brillante|Clair|Hell
custom|Custom|自定义|自訂|カスタム|Personalizado|Personnalisé|Benutzerdefiniert
favorite|Favorite|收藏|收藏|お気に入りに追加|Añadir a favoritos|Ajouter aux favoris|Favorisieren
unfavorite|Unfavorite|取消收藏|取消收藏|お気に入りを解除|Quitar de favoritos|Retirer des favoris|Favorit entfernen
shareTrack|Share to chat|分享到群聊|分享到群聊|チャットに共有|Compartir en el chat|Partager dans la discussion|Im Chat teilen
queueSong|Queue song|点播|點播|曲を追加|Añadir canción|Ajouter le morceau|Titel einreihen
queued|Queued {title}|已点播 {title}|已點播 {title}|{title} を追加しました|Añadida: {title}|Ajouté : {title}|Eingereiht: {title}
playNext|Play next|下一首播放|下一首播放|次に再生|Reproducir después|Lire ensuite|Als Nächstes spielen
quota|{count} requests remaining|还可点 {count} 首|還可點 {count} 首|あと {count} 曲追加可能|Quedan {count} solicitudes|{count} demandes restantes|Noch {count} Musikwünsche
shuffle|Shuffle queue|打乱待播|打亂待播|待ちリストをシャッフル|Mezclar cola|Mélanger la file|Warteschlange mischen
randomFill|Random fill|随机补充|隨機補充|ランダム追加|Rellenar al azar|Remplir au hasard|Zufällig auffüllen
top|Pin to top|置顶|置頂|先頭に移動|Fijar arriba|Placer en tête|Nach oben setzen
moveUp|Move up|上移|上移|上へ移動|Subir|Monter|Nach oben
moveDown|Move down|下移|下移|下へ移動|Bajar|Descendre|Nach unten
reorder|Drag to reorder|拖动排序|拖曳排序|ドラッグして並べ替え|Arrastrar para ordenar|Glisser pour réordonner|Zum Sortieren ziehen
requeue|Re-queue|再次点播|再次點播|もう一度追加|Volver a añadir|Remettre en file|Erneut einreihen
olderHistory|Load older history|加载更早历史|載入更早歷史|以前の履歴を読み込む|Cargar historial anterior|Charger l’historique précédent|Älteren Verlauf laden
songs|Songs|歌曲|歌曲|曲|Canciones|Morceaux|Titel
albums|Albums|专辑|專輯|アルバム|Álbumes|Albums|Alben
artists|Artists|歌手|歌手|アーティスト|Artistas|Artistes|Künstler
playlists|Playlists|歌单|歌單|プレイリスト|Listas|Listes de lecture|Wiedergabelisten
search|Search|搜索|搜尋|検索|Buscar|Rechercher|Suchen
searchLibrary|Search library|搜索曲库|搜尋曲庫|ライブラリを検索|Buscar en biblioteca|Rechercher dans la bibliothèque|Bibliothek durchsuchen
searchPlaceholder|Songs, albums or artists|搜索歌曲、专辑或歌手|搜尋歌曲、專輯或歌手|曲・アルバム・アーティスト|Canciones, álbumes o artistas|Morceaux, albums ou artistes|Titel, Alben oder Künstler
searchResults|Results for “{query}”|“{query}”的搜索结果|「{query}」的搜尋結果|「{query}」の検索結果|Resultados para «{query}»|Résultats pour « {query} »|Ergebnisse für „{query}“
clear|Clear|清空|清空|クリア|Borrar|Effacer|Leeren
refresh|Refresh|刷新|重新整理|更新|Actualizar|Actualiser|Aktualisieren
randomRefresh|Refresh random|刷新随机歌曲|重新整理隨機歌曲|ランダム曲を更新|Renovar selección aleatoria|Renouveler la sélection|Zufallsauswahl erneuern
lucky|I’m feeling lucky|手气不错|手氣不錯|おまかせで一曲|Voy a tener suerte|J’ai de la chance|Auf gut Glück
back|Back to results|返回结果|返回結果|結果に戻る|Volver a resultados|Retour aux résultats|Zurück zu Ergebnissen
queueAll|Queue all|整张点播|整張點播|すべて追加|Añadir todo|Tout ajouter|Alle einreihen
openAlbum|Open album|打开专辑|開啟專輯|アルバムを開く|Abrir álbum|Ouvrir l’album|Album öffnen
openArtist|Open artist|打开歌手|開啟歌手|アーティストを開く|Abrir artista|Ouvrir l’artiste|Künstler öffnen
unknownSong|Unknown song|未知歌曲|未知歌曲|不明な曲|Canción desconocida|Morceau inconnu|Unbekannter Titel
unknownArtist|Unknown artist|未知歌手|未知歌手|不明なアーティスト|Artista desconocido|Artiste inconnu|Unbekannter Künstler
unknownAlbum|Unknown album|未知专辑|未知專輯|不明なアルバム|Álbum desconocido|Album inconnu|Unbekanntes Album
members|Members|成员|成員|メンバー|Miembros|Membres|Mitglieder
online|Online|在线|在線|オンライン|En línea|En ligne|Online
offline|Offline|离线|離線|オフライン|Sin conexión|Hors ligne|Offline
owner|Owner|房主|房主|オーナー|Propietario|Propriétaire|Besitzer
manager|Manager|管理员|管理員|管理者|Administrador|Gestionnaire|Verwalter
member|Member|成员|成員|メンバー|Miembro|Membre|Mitglied
statistics|Statistics & rankings|统计与排行|統計與排行|統計とランキング|Estadísticas y clasificaciones|Statistiques et classements|Statistiken und Ranglisten
wishes|Wishes|心愿|心願|ウィッシュ|Deseos|Souhaits|Wünsche
day|Day|日|日|日|Día|Jour|Tag
week|Week|周|週|週|Semana|Semaine|Woche
month|Month|月|月|月|Mes|Mois|Monat
year|Year|年|年|年|Año|Année|Jahr
all|All time|全部|全部|全期間|Todo|Tout|Gesamt
plays|Plays|播放次数|播放次數|再生回数|Reproducciones|Lectures|Wiedergaben
listeningTime|Listening time|收听时长|收聽時長|聴取時間|Tiempo de escucha|Temps d’écoute|Hörzeit
chatCount|Messages|聊天数|聊天數|メッセージ数|Mensajes|Messages|Nachrichten
activeMembers|Active members|活跃成员|活躍成員|アクティブメンバー|Miembros activos|Membres actifs|Aktive Mitglieder
topDJs|Top DJs|点歌达人|點歌達人|トップ DJ|Mejores DJ|Meilleurs DJ|Top-DJs
topListeners|Top listeners|收听达人|收聽達人|トップリスナー|Más oyentes|Meilleurs auditeurs|Top-Zuhörer
topChatters|Top chatters|聊天达人|聊天達人|トップチャッター|Más conversadores|Plus bavards|Top-Chatter
topArtists|Top artists|热门歌手|熱門歌手|人気アーティスト|Artistas destacados|Artistes populaires|Top-Künstler
topAlbums|Top albums|热门专辑|熱門專輯|人気アルバム|Álbumes destacados|Albums populaires|Top-Alben
topTracks|Top tracks|热门歌曲|熱門歌曲|人気曲|Canciones destacadas|Morceaux populaires|Top-Titel
activity|24-hour listening activity|24 小时收听时钟|24 小時收聽時鐘|24 時間の聴取状況|Actividad de escucha de 24 horas|Activité d’écoute sur 24 heures|Höraktivität über 24 Stunden
minutes|{count} min|{count} 分钟|{count} 分鐘|{count} 分|{count} min|{count} min|{count} Min.
pending|Pending|待实现|待實現|未対応|Pendiente|En attente|Offen
fulfilled|Fulfilled|已实现|已實現|対応済み|Cumplido|Réalisé|Erfüllt
newWish|Make a wish|许个愿|許個願|ウィッシュを作成|Pedir un deseo|Faire un souhait|Wunsch erstellen
track|Track|歌曲|歌曲|曲|Canción|Morceau|Titel
album|Album|专辑|專輯|アルバム|Álbum|Album|Album
artist|Artist|歌手|歌手|アーティスト|Artista|Artiste|Künstler
kind|Type|类型|類型|種類|Tipo|Type|Typ
title|Title|标题|標題|タイトル|Título|Titre|Titel
reason|Reason (optional)|理由（可选）|理由（選填）|理由（任意）|Motivo (opcional)|Raison (facultatif)|Grund (optional)
submit|Submit|提交|提交|送信|Enviar|Envoyer|Absenden
fulfill|Fulfill wish|实现心愿|實現心願|ウィッシュに対応|Cumplir deseo|Réaliser le souhait|Wunsch erfüllen
response|Response note|回复说明|回覆說明|返信メモ|Nota de respuesta|Note de réponse|Antwortnotiz
attachResource|Select matching resources|选择匹配的曲库资源|選擇符合的曲庫資源|該当するリソースを選択|Selecciona recursos coincidentes|Sélectionnez les ressources correspondantes|Passende Ressourcen auswählen
level|Level {level} / 10|等级 {level} / 10|等級 {level} / 10|レベル {level} / 10|Nivel {level} / 10|Niveau {level} / 10|Stufe {level} / 10
nextLevel|{count} minutes to the next level|距下一级 {count} 分钟|距下一級 {count} 分鐘|次のレベルまで {count} 分|{count} minutos para el siguiente nivel|{count} minutes avant le prochain niveau|{count} Minuten bis zur nächsten Stufe
maxLevel|Maximum level reached|已达最高等级|已達最高等級|最高レベルに到達|Nivel máximo alcanzado|Niveau maximum atteint|Höchste Stufe erreicht
listened|Tracks heard|听过歌曲|聽過歌曲|聴いた曲|Canciones escuchadas|Morceaux écoutés|Gehörte Titel
requested|Tracks requested|点播歌曲|點播歌曲|リクエスト曲|Canciones pedidas|Morceaux demandés|Gewünschte Titel
stayTime|Time in room|在房时长|在房時長|滞在時間|Tiempo en la sala|Temps dans le salon|Zeit im Raum
badges|Achievements|成就|成就|実績|Logros|Succès|Erfolge
badge|Achievement badge|成就徽章|成就徽章|実績バッジ|Insignia de logro|Badge de succès|Erfolgsabzeichen
unlocked|Unlocked|已解锁|已解鎖|解除済み|Desbloqueado|Débloqué|Freigeschaltet
locked|In progress|进行中|進行中|進行中|En progreso|En cours|In Arbeit
selected|Selected|已选择|已選擇|選択中|Seleccionado|Sélectionné|Ausgewählt
clearBadge|Clear display badge|清除展示徽章|清除展示徽章|表示バッジを解除|Quitar insignia visible|Retirer le badge affiché|Angezeigtes Abzeichen entfernen
mention|@Mention in chat|@提及到群聊|@提及到群聊|チャットで @メンション|@Mencionar en el chat|@Mentionner dans la discussion|Im Chat @erwähnen
roomName|Room name|房间名称|房間名稱|ルーム名|Nombre de sala|Nom du salon|Raumname
allowSkip|Allow members to skip tracks|允许成员跳歌|允許成員跳歌|メンバーのスキップを許可|Permitir saltos a miembros|Autoriser les membres à passer|Mitgliedern Überspringen erlauben
playbackMode|Playback mode|播放模式|播放模式|再生モード|Modo de reproducción|Mode de lecture|Wiedergabemodus
fifo|First in, first out|先进先播|先進先播|追加順|Por orden de llegada|Ordre d’arrivée|Nach Reihenfolge
fair_random|Fair random|公平随机|公平隨機|公平ランダム|Aleatorio equitativo|Aléatoire équitable|Fairer Zufall
queueLimit|Requests per member|每人点歌上限|每人點歌上限|一人あたりのリクエスト上限|Peticiones por miembro|Demandes par membre|Wünsche pro Mitglied
preload|Preload next track|预加载下一首|預載下一首|次の曲を事前読み込み|Precargar siguiente canción|Précharger le morceau suivant|Nächsten Titel vorladen
admin|Admin console|管理控制台|管理控制台|管理コンソール|Consola de administración|Console d’administration|Verwaltung
shareRoom|Share room|分享房间|分享房間|ルームを共有|Compartir sala|Partager le salon|Raum teilen
shareInviteHint|This invitation stays available until you refresh this page.|刷新页面前可继续分享本次邀请。|重新整理頁面前可繼續分享本次邀請。|ページを更新するまで招待を共有できます。|Puedes compartir esta invitación hasta recargar la página.|Cette invitation reste disponible jusqu’au rechargement.|Diese Einladung ist bis zum Neuladen verfügbar.
shareMemberHint|This link is for existing members. Create invitations in the admin console.|此链接用于已有成员；新邀请可在管理页创建。|此連結供現有成員使用；新邀請可於管理頁建立。|既存メンバー用リンクです。招待は管理画面で作成できます。|Enlace para miembros actuales. Crea invitaciones en la consola.|Lien pour les membres existants. Créez les invitations dans la console.|Link für bestehende Mitglieder. Einladungen gibt es in der Verwaltung.
copy|Copy link|复制链接|複製連結|リンクをコピー|Copiar enlace|Copier le lien|Link kopieren
copied|Copied|已复制|已複製|コピーしました|Copiado|Copié|Kopiert
musicMate|Open in MusicMate|在 MusicMate 中打开|在 MusicMate 中開啟|MusicMate で開く|Abrir en MusicMate|Ouvrir dans MusicMate|In MusicMate öffnen
olderMessages|Load earlier messages|加载更早消息|載入更早訊息|以前のメッセージを読み込む|Cargar mensajes anteriores|Charger les messages précédents|Ältere Nachrichten laden
messagePlaceholder|Say something… @ a member or song|聊点什么… @成员或歌曲|聊點什麼… @成員或歌曲|メッセージ… @メンバーや曲|Escribe algo… @miembro o canción|Écrivez… @membre ou morceau|Schreib etwas… @Mitglied oder Titel
send|Send|发送|傳送|送信|Enviar|Envoyer|Senden
reply|Reply|回复|回覆|返信|Responder|Répondre|Antworten
recall|Recall (within 5 min)|撤回（5 分钟内）|收回（5 分鐘內）|取り消す（5 分以内）|Retirar (en 5 min)|Retirer (sous 5 min)|Zurückrufen (innerhalb 5 Min.)
recalled|Message recalled|消息已撤回|訊息已收回|メッセージは取り消されました|Mensaje retirado|Message retiré|Nachricht zurückgerufen
deleted|Message deleted|消息已删除|訊息已刪除|メッセージは削除されました|Mensaje eliminado|Message supprimé|Nachricht gelöscht
stickers|Stickers|贴纸|貼圖|ステッカー|Pegatinas|Autocollants|Sticker
sticker|Sticker {emoji}|贴纸 {emoji}|貼圖 {emoji}|ステッカー {emoji}|Pegatina {emoji}|Autocollant {emoji}|Sticker {emoji}
stickerPack|Sticker pack {count}|贴纸包 {count}|貼圖包 {count}|ステッカーパック {count}|Paquete {count}|Pack {count}|Stickerpaket {count}
quickReviews|Quick reviews|歌曲快评|歌曲快評|クイック感想|Opiniones rápidas|Avis rapides|Kurze Bewertungen
reviewLove|Love this song!|这首太好听了！|這首太好聽了！|この曲、大好き！|¡Me encanta esta canción!|J’adore ce morceau !|Ich liebe diesen Titel!
reviewVibe|Perfect vibe.|氛围感拉满。|氛圍感滿滿。|最高の雰囲気。|Ambiente perfecto.|Ambiance parfaite.|Perfekte Stimmung.
reviewAgain|One more time!|再听一遍！|再聽一遍！|もう一回聴きたい！|¡Una vez más!|Encore une fois !|Noch einmal!
pack.vibes|Music vibes|音乐氛围|音樂氛圍|音楽の気分|Ambiente musical|Ambiance musicale|Musikstimmung
pack.reactions|Reactions|心情回应|心情回應|リアクション|Reacciones|Réactions|Reaktionen
badge.requested|Request {count} tracks|点播 {count} 首歌曲|點播 {count} 首歌曲|{count} 曲リクエスト|Pedir {count} canciones|Demander {count} morceaux|{count} Titel wünschen
badge.listened|Listen to {count} tracks|收听 {count} 首歌曲|收聽 {count} 首歌曲|{count} 曲を聴く|Escuchar {count} canciones|Écouter {count} morceaux|{count} Titel hören
badge.chat|Send {count} messages|发送 {count} 条消息|傳送 {count} 則訊息|{count} 件のメッセージ|Enviar {count} mensajes|Envoyer {count} messages|{count} Nachrichten senden
badge.artist|Artist fan · {count} plays|歌手粉丝 · {count} 次|歌手粉絲 · {count} 次|アーティストファン · {count} 回|Fan del artista · {count} escuchas|Fan de l’artiste · {count} écoutes|Künstlerfan · {count} Wiedergaben
badge.album|Album fan · {count} plays|专辑粉丝 · {count} 次|專輯粉絲 · {count} 次|アルバムファン · {count} 回|Fan del álbum · {count} escuchas|Fan de l’album · {count} écoutes|Albumfan · {count} Wiedergaben
error.chat_recall_expired|Messages can only be recalled within five minutes.|只能撤回五分钟内的消息。|只能收回五分鐘內的訊息。|メッセージを取り消せるのは 5 分以内です。|Solo se pueden retirar mensajes en cinco minutos.|Les messages peuvent être retirés sous cinq minutes.|Nachrichten können nur innerhalb von fünf Minuten zurückgerufen werden.
error.wish_limit_reached|You already have 50 pending wishes.|你已有 50 个待实现心愿。|你已有 50 個待實現心願。|未対応のウィッシュが 50 件あります。|Ya tienes 50 deseos pendientes.|Vous avez déjà 50 souhaits en attente.|Du hast bereits 50 offene Wünsche.
error.badge_locked|This badge has not been unlocked yet.|此徽章尚未解锁。|此徽章尚未解鎖。|このバッジはまだ解除されていません。|Esta insignia aún está bloqueada.|Ce badge n’est pas encore débloqué.|Dieses Abzeichen ist noch nicht freigeschaltet.
error.generic|Something went wrong. Please retry.|操作失败，请重试。|操作失敗，請重試。|エラーが発生しました。再試行してください。|Algo salió mal. Reinténtalo.|Une erreur est survenue. Réessayez.|Etwas ist schiefgelaufen. Bitte erneut versuchen.
error.room_closed|This room is closed.|房间已关闭。|房間已關閉。|ルームは閉鎖されています。|Esta sala está cerrada.|Ce salon est fermé.|Dieser Raum ist geschlossen.
error.membership_required|Join with a valid invitation first.|请先使用有效邀请加入房间。|請先使用有效邀請加入房間。|有効な招待で参加してください。|Entra primero con una invitación válida.|Rejoignez avec une invitation valide.|Bitte mit einer gültigen Einladung beitreten.
error.library_access_required|You do not have access to this music library.|你没有此音乐库的访问权限。|你沒有此音樂庫的存取權限。|このライブラリにアクセスできません。|No tienes acceso a esta biblioteca.|Vous n’avez pas accès à cette bibliothèque.|Du hast keinen Zugriff auf diese Bibliothek.
error.personal_queue_limit|Your queue is full. Wait for a track to play.|点歌额度已满，请等待歌曲播放。|點歌額度已滿，請等待歌曲播放。|追加上限です。再生をお待ちください。|Tu cola está llena. Espera a que se reproduzca una canción.|Votre file est pleine. Attendez la lecture d’un morceau.|Deine Warteschlange ist voll. Warte auf die Wiedergabe.
error.room_manager_required|Only room managers can do this.|此操作需要房间管理权限。|此操作需要房間管理權限。|管理権限が必要です。|Solo los administradores pueden hacerlo.|Cette action nécessite les droits de gestion.|Hierfür sind Verwaltungsrechte erforderlich.
error.skip_permission_required|You cannot skip this track.|你无权跳过此歌曲。|你無權跳過此歌曲。|この曲はスキップできません。|No puedes saltar esta canción.|Vous ne pouvez pas passer ce morceau.|Du darfst diesen Titel nicht überspringen.
error.revision_conflict|Playback changed. The room has been refreshed.|播放状态已变化，已刷新房间。|播放狀態已變化，已重新整理房間。|再生状態が変わりました。更新しました。|La reproducción cambió. Sala actualizada.|La lecture a changé. Le salon a été actualisé.|Die Wiedergabe hat sich geändert. Der Raum wurde aktualisiert.
error.navidrome_login_failed|Incorrect Navidrome username or password.|Navidrome 用户名或密码错误。|Navidrome 使用者名稱或密碼錯誤。|Navidrome の認証情報が正しくありません。|Usuario o contraseña de Navidrome incorrectos.|Identifiant ou mot de passe Navidrome incorrect.|Navidrome-Benutzername oder Passwort falsch.
error.session_required|Your session expired. Please sign in again.|会话已过期，请重新登录。|工作階段已過期，請重新登入。|セッションが切れました。再度ログインしてください。|Sesión caducada. Inicia sesión de nuevo.|Session expirée. Reconnectez-vous.|Sitzung abgelaufen. Bitte erneut anmelden.
error.room_id_invalid|This room link is invalid.|房间链接无效。|房間連結無效。|ルームリンクが無効です。|El enlace de la sala no es válido.|Ce lien de salon est invalide.|Dieser Raumlink ist ungültig.
error.same_origin_required|The room and Navidrome must use the same public domain.|房间和 Navidrome 必须使用相同的公开域名。|房間和 Navidrome 必須使用相同的公開網域。|ルームと Navidrome は同一ドメインが必要です。|La sala y Navidrome deben usar el mismo dominio.|Le salon et Navidrome doivent partager le même domaine.|Raum und Navidrome müssen dieselbe öffentliche Domain nutzen.
error.track_not_found|This track is unavailable.|此歌曲不可用。|此歌曲無法使用。|この曲は利用できません。|Esta canción no está disponible.|Ce morceau est indisponible.|Dieser Titel ist nicht verfügbar.
error.rate_limited|Too many requests. Please wait a moment.|请求过于频繁，请稍后再试。|請求過於頻繁，請稍後再試。|リクエストが多すぎます。しばらくお待ちください。|Demasiadas solicitudes. Espera un momento.|Trop de demandes. Patientez un instant.|Zu viele Anfragen. Bitte kurz warten.
error.invite_invalid|This invitation is invalid or expired.|邀请无效或已过期。|邀請無效或已過期。|招待は無効か期限切れです。|La invitación no es válida o ha caducado.|Invitation invalide ou expirée.|Die Einladung ist ungültig oder abgelaufen.
error.feature_unavailable|This feature requires the updated room server.|此功能需要更新后的房间服务器。|此功能需要更新後的房間伺服器。|この機能にはサーバーの更新が必要です。|Esta función requiere actualizar el servidor.|Cette fonction nécessite un serveur à jour.|Diese Funktion erfordert den aktualisierten Raumserver.
error.eq_unavailable|This browser cannot enable EQ.|此浏览器无法启用均衡器。|此瀏覽器無法啟用等化器。|このブラウザーでは EQ を有効にできません。|Este navegador no puede activar el EQ.|Ce navigateur ne peut pas activer l’égaliseur.|Dieser Browser kann EQ nicht aktivieren.
error.request_invalid|Please check the submitted values.|请检查填写的内容。|請檢查填寫的內容。|入力内容を確認してください。|Comprueba los datos introducidos.|Vérifiez les valeurs saisies.|Bitte die Eingaben prüfen.
error.network|Connection failed. Please retry.|连接失败，请重试。|連線失敗，請重試。|接続に失敗しました。再試行してください。|Error de conexión. Reinténtalo.|Échec de connexion. Réessayez.|Verbindung fehlgeschlagen. Bitte erneut versuchen.
`.trim().split('\n').map(row => row.split('|'))

export const dictionaries = Object.fromEntries(locales.map((code, index) => [code,
  Object.fromEntries(messageRows.map(([key, ...messages]) => [key, messages[index]])),
])) as Record<Locale, Record<string, string>>

export function mapLocale(value: string | null | undefined): Locale | undefined {
  const tag = (value || '').trim().replace(/_/g, '-').toLowerCase()
  if (/^zh(?:-|$)/.test(tag)) return /(?:^|-)tw(?:-|$)|(?:^|-)hk(?:-|$)|(?:^|-)hant(?:-|$)/.test(tag) ? 'zh-TW' : 'zh-CN'
  return locales.find(code => code === tag.split('-')[0])
}
export function resolveLocale(
  storage: Pick<Storage, 'getItem'> | undefined = undefined,
  browser: Pick<Navigator, 'language' | 'languages'> = globalThis.navigator,
): Locale {
  for (const key of ['nmr:locale', 'i18nextLng', 'locale']) {
    try { const match = mapLocale((storage ?? globalThis.localStorage)?.getItem(key)); if (match) return match } catch { /* Storage can be disabled. */ }
  }
  for (const language of [...(browser?.languages || []), browser?.language]) {
    const match = mapLocale(language)
    if (match) return match
  }
  return 'en'
}
const currentLocale = ref<Locale>(resolveLocale())
export function setLocale(value: Locale): void {
  currentLocale.value = value
  try { localStorage.setItem('nmr:locale', value) } catch { /* Keep the in-memory choice. */ }
}
export function translate(locale: Locale, key: string, params: Record<string, string | number> = {}): string {
  const message = dictionaries[locale]?.[key] ?? dictionaries.en[key] ?? key
  return message.replace(/\{(\w+)\}/g, (token, name: string) => params[name] === undefined ? token : String(params[name]))
}
export function t(key: string, params?: Record<string, string | number>): string {
  return translate(currentLocale.value, key, params)
}
const errorAliases: Record<string, string> = {
  recall_expired: 'chat_recall_expired', message_sender_required: 'room_manager_required',
  wish_author_required: 'room_manager_required', wish_not_found: 'request_invalid',
  chat_invalid: 'request_invalid', chat_empty: 'request_invalid', sticker_invalid: 'request_invalid', mention_invalid: 'request_invalid',
  reference_invalid: 'request_invalid', message_id_invalid: 'request_invalid', wish_invalid: 'request_invalid',
  wish_response_invalid: 'request_invalid', wish_resource_invalid: 'request_invalid', wish_status_invalid: 'request_invalid',
  wish_id_invalid: 'request_invalid', period_invalid: 'request_invalid', message_not_found: 'request_invalid',
  chat_owner_required: 'room_manager_required', message_owner_required: 'room_manager_required',
  login_failed: 'navidrome_login_failed', navidrome_auth_failed: 'navidrome_login_failed',
  auth_proof_invalid: 'navidrome_login_failed', session_missing: 'session_required', session_expired: 'session_required',
  session_refresh_failed: 'session_required', ws_ticket_invalid: 'session_required',
  plugin_user_not_allowed: 'membership_required', navidrome_admin_required: 'room_manager_required',
  queue_entry_owner_required: 'room_manager_required', wish_owner_required: 'room_manager_required',
  room_not_found: 'room_id_invalid', track_invalid: 'track_not_found',
  invite_expired: 'invite_invalid', invite_revoked: 'invite_invalid', invite_exhausted: 'invite_invalid',
  auth_rate_limited: 'rate_limited', mutation_rate_limited: 'rate_limited', chat_rate_limited: 'rate_limited',
  room_name_invalid: 'request_invalid', queue_limit_invalid: 'request_invalid', playback_mode_invalid: 'request_invalid',
  queue_order_invalid: 'request_invalid', queue_id_invalid: 'request_invalid',
  feature_not_implemented: 'feature_unavailable', not_implemented: 'feature_unavailable',
  navidrome_unavailable: 'network', subsonic_failed: 'network', database_unavailable: 'network',
}
export function errorText(cause: unknown): string {
  const code = typeof cause === 'string' ? cause : cause && typeof cause === 'object' && 'code' in cause ? String(cause.code) : 'network'
  const key = `error.${errorAliases[code] || code}`
  return t(dictionaries.en[key] ? key : 'error.generic')
}
export function useI18n() { return { locale: readonly(currentLocale), locales, localeNames, setLocale, t, errorText } }
