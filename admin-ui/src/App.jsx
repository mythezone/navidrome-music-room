import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  AppBar,
  Box,
  Button,
  Card,
  CardContent,
  Checkbox,
  Chip,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  FormControl,
  FormControlLabel,
  FormGroup,
  Grid,
  IconButton,
  InputLabel,
  List,
  ListItem,
  ListItemIcon,
  ListItemSecondaryAction,
  ListItemText,
  MenuItem,
  Paper,
  Select,
  Snackbar,
  Step,
  StepLabel,
  Stepper,
  TextField,
  Toolbar,
  Tooltip,
  Typography,
  makeStyles,
} from '@material-ui/core'
import {
  Add as AddIcon,
  ArrowBack as ArrowBackIcon,
  CheckCircle as CheckCircleIcon,
  Close as CloseIcon,
  Code as CodeIcon,
  DeleteOutline as DeleteIcon,
  Edit as EditIcon,
  Group as GroupIcon,
  Headset as HeadsetIcon,
  Link as LinkIcon,
  Lock as LockIcon,
  Language as LanguageIcon,
  SystemUpdateAlt as SystemUpdateIcon,
  History as HistoryIcon,
  GetApp as DownloadIcon,
  Warning as WarningIcon,
  MeetingRoom as RoomIcon,
  PowerSettingsNew as PowerIcon,
  Refresh as RefreshIcon,
  Share as ShareIcon,
  Web as WebIcon,
} from '@material-ui/icons'
import { QRCodeSVG } from 'qrcode.react'
import { GatewayAPI, hasCompleteProof, readNavidromeProof } from './api'
import { detectLocale, formatDate, locales, localizeError, translate } from './i18n'
import { runUpgrade, sameVersion, waitForReady } from './updates'

const useStyles = makeStyles((theme) => ({
  root: { minHeight: '100vh', background: 'linear-gradient(160deg, #303030 0%, #263238 100%)' },
  appBar: { background: '#212121', color: '#fff', boxShadow: '0 2px 10px rgba(0,0,0,.35)' },
  toolbar: { minHeight: 64, gap: theme.spacing(1.5), [theme.breakpoints.down('xs')]: { gap: theme.spacing(.5), paddingLeft: theme.spacing(1), paddingRight: theme.spacing(1) } },
  brand: { display: 'flex', alignItems: 'center', gap: theme.spacing(1), flexGrow: 1, minWidth: 0 },
  brandTitle: { whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' },
  desktopOnly: { [theme.breakpoints.down('xs')]: { display: 'none' } },
  content: { maxWidth: 1440, margin: '0 auto', padding: theme.spacing(3), [theme.breakpoints.down('xs')]: { padding: theme.spacing(1.5) } },
  statusCard: { marginBottom: theme.spacing(2), background: 'rgba(66,66,66,.82)', border: '1px solid rgba(255,255,255,.08)' },
  statusRow: { display: 'flex', alignItems: 'center', gap: theme.spacing(1), flexWrap: 'wrap' },
  panel: { minHeight: 610, overflow: 'hidden', border: '1px solid rgba(255,255,255,.08)', [theme.breakpoints.down('sm')]: { minHeight: 0 } },
  panelHeader: { minHeight: 72, padding: theme.spacing(2, 2.5), display: 'flex', alignItems: 'center', gap: theme.spacing(1.5) },
  grow: { flexGrow: 1 },
  roomItem: { borderBottom: '1px solid rgba(255,255,255,.07)', padding: theme.spacing(1.5, 2) },
  selectedRoom: { background: 'rgba(144,202,249,.13)' },
  roomAvatar: {
    width: 42,
    height: 42,
    display: 'grid',
    placeItems: 'center',
    borderRadius: 8,
    background: 'rgba(144,202,249,.14)',
    color: theme.palette.primary.main,
  },
  empty: { minHeight: 450, display: 'grid', placeItems: 'center', textAlign: 'center', padding: theme.spacing(4), color: theme.palette.text.secondary },
  detailBody: { padding: theme.spacing(2.5) },
  detailActions: { display: 'flex', flexWrap: 'wrap', gap: theme.spacing(1), marginTop: theme.spacing(2) },
  metric: { padding: theme.spacing(1.5), background: 'rgba(0,0,0,.12)', borderRadius: 8, overflowWrap: 'anywhere' },
  section: { marginTop: theme.spacing(3) },
  sectionHeader: { display: 'flex', alignItems: 'center', gap: theme.spacing(1), marginBottom: theme.spacing(1) },
  onlineDot: { width: 8, height: 8, borderRadius: '50%', background: theme.palette.success.main },
  offlineDot: { width: 8, height: 8, borderRadius: '50%', background: theme.palette.grey[600] },
  qrBox: { display: 'grid', placeItems: 'center', background: '#fff', padding: theme.spacing(2), borderRadius: 10, width: 'fit-content', margin: `${theme.spacing(2)}px auto` },
  shareLink: { fontFamily: 'ui-monospace, SFMono-Regular, Menlo, monospace', overflowWrap: 'anywhere', fontSize: '.78rem', padding: theme.spacing(1.5), borderRadius: 6, background: 'rgba(0,0,0,.22)' },
  roadmap: { borderColor: 'rgba(99,223,209,.24)', background: 'rgba(99,223,209,.035)' },
  loginCard: { maxWidth: 560, margin: `${theme.spacing(10)}px auto`, padding: theme.spacing(4), textAlign: 'center' },
  loading: { minHeight: 460, display: 'grid', placeItems: 'center' },
  danger: { color: theme.palette.error.light },
  snackbar: { '& .MuiSnackbarContent-root': { background: '#323232', color: '#fff' } },
}))

async function copyText(value) {
  if (navigator.clipboard && window.isSecureContext) return navigator.clipboard.writeText(value)
  const node = document.createElement('textarea')
  node.value = value
  node.style.position = 'fixed'
  node.style.opacity = '0'
  document.body.appendChild(node)
  node.select()
  try {
    if (!document.execCommand('copy')) return Promise.reject({ code: 'copy_failed' })
    return Promise.resolve()
  } finally {
    node.remove()
  }
}

function RoomDialog({ open, room, folders, busy, onClose, onSave, t }) {
  const [name, setName] = useState('')
  const [queueLimit, setQueueLimit] = useState(20)
  const [playbackMode, setPlaybackMode] = useState('fifo')
  const [preload, setPreload] = useState(true)
  const [allowMemberSkip, setAllowMemberSkip] = useState(true)
  const [selectedFolders, setSelectedFolders] = useState([])

  useEffect(() => {
    setName(room?.name || '')
    setQueueLimit(room?.queueLimit || 20)
    setPlaybackMode(room?.playbackMode || 'fifo')
    setPreload(room?.preloadNextTrack ?? true)
    setAllowMemberSkip(room?.allowMemberSkip ?? true)
    setSelectedFolders(room?.musicFolderIDs || folders.map((folder) => folder.id))
  }, [room, folders, open])

  const submit = (event) => {
    event.preventDefault()
    onSave({
      name: name.trim(),
      queueLimit: Number(queueLimit),
      playbackMode,
      musicFolderIDs: selectedFolders.map(Number),
      preloadNextTrack: preload,
      allowMemberSkip,
    })
  }

  return (
    <Dialog open={open} onClose={busy ? undefined : onClose} fullWidth maxWidth="sm" aria-labelledby="room-dialog-title">
      <form onSubmit={submit}>
        <DialogTitle id="room-dialog-title">{t(room ? 'roomSettings' : 'createRoom')}</DialogTitle>
        <DialogContent dividers>
          <TextField id="room-name" label={t('roomName')} value={name} onChange={(event) => setName(event.target.value)} fullWidth required inputProps={{ maxLength: 80 }} margin="normal" autoFocus />
          <Grid container spacing={2}>
            <Grid item xs={12} sm={6}>
              <TextField id="room-queue-limit" label={t('queueLimit')} type="number" value={queueLimit} onChange={(event) => setQueueLimit(event.target.value)} fullWidth margin="normal" inputProps={{ min: 1, max: 100 }} />
            </Grid>
            <Grid item xs={12} sm={6}>
              <FormControl fullWidth margin="normal">
                <InputLabel id="playback-mode-label">{t('queueMode')}</InputLabel>
                <Select labelId="playback-mode-label" value={playbackMode} onChange={(event) => setPlaybackMode(event.target.value)}>
                  <MenuItem value="fifo">{t('fifo')}</MenuItem>
                  <MenuItem value="fair_random">{t('fair_random')}</MenuItem>
                </Select>
              </FormControl>
            </Grid>
          </Grid>
          <Typography variant="subtitle2">{t('libraries')}</Typography>
          <FormGroup>
            {folders.map((folder) => (
              <FormControlLabel
                key={folder.id}
                control={<Checkbox color="primary" checked={selectedFolders.includes(folder.id)} onChange={(event) => setSelectedFolders((current) => event.target.checked ? [...current, folder.id] : current.filter((id) => id !== folder.id))} />}
                label={folder.name || t('musicFolder', { id: folder.id })}
              />
            ))}
          </FormGroup>
          {!folders.length && <Typography variant="body2" color="textSecondary">{t('noLibraries')}</Typography>}
          <FormControlLabel control={<Checkbox color="primary" checked={preload} onChange={(event) => setPreload(event.target.checked)} />} label={t('preload')} />
          <FormControlLabel control={<Checkbox color="primary" checked={allowMemberSkip} onChange={(event) => setAllowMemberSkip(event.target.checked)} />} label={t('allowMemberSkip')} />
        </DialogContent>
        <DialogActions>
          <Button onClick={onClose} disabled={busy}>{t('cancel')}</Button>
          <Button color="primary" variant="contained" type="submit" disabled={busy || !name.trim() || !selectedFolders.length}>
            {busy ? <CircularProgress size={20} aria-label={t('loading')} /> : t(room ? 'save' : 'createRoom')}
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  )
}

function ShareDialog({ open, room, busy, onClose, onCreate, onCopied, onError, t }) {
  const classes = useStyles()
  const [label, setLabel] = useState('')
  const [days, setDays] = useState(7)
  const [maxUses, setMaxUses] = useState(20)
  const [singleUse, setSingleUse] = useState(false)
  const [created, setCreated] = useState(null)

  useEffect(() => {
    if (open) {
      setCreated(null)
      setLabel('')
      setDays(7)
      setMaxUses(20)
      setSingleUse(false)
    }
  }, [open, room?.roomID])

  const create = async () => {
    const expiresAt = new Date(Date.now() + Number(days) * 86400000).toISOString()
    const invite = await onCreate({ label: label.trim() || t('shareInvite'), expiresAt, maxUses: singleUse ? 1 : Number(maxUses), singleUse })
    if (invite) setCreated(invite)
  }

  return (
    <Dialog open={open} onClose={busy ? undefined : onClose} fullWidth maxWidth="sm" aria-labelledby="share-dialog-title">
      <DialogTitle id="share-dialog-title">{t('shareTitle', { name: room?.name || '' })}</DialogTitle>
      <DialogContent dividers>
        {!created ? (
          <>
            <Typography color="textSecondary" gutterBottom>{t('inviteHelp')}</Typography>
            <TextField id="invite-label" label={t('inviteLabel')} value={label} placeholder={t('shareInvite')} onChange={(event) => setLabel(event.target.value)} fullWidth margin="normal" inputProps={{ maxLength: 80 }} />
            <Grid container spacing={2}>
              <Grid item xs={6}>
                <TextField id="invite-days" label={t('inviteDays')} type="number" value={days} onChange={(event) => setDays(event.target.value)} fullWidth margin="normal" inputProps={{ min: 1, max: 365 }} />
              </Grid>
              <Grid item xs={6}>
                <TextField id="invite-max-uses" label={t('maxUses')} type="number" value={singleUse ? 1 : maxUses} onChange={(event) => setMaxUses(event.target.value)} fullWidth margin="normal" disabled={singleUse} inputProps={{ min: 1, max: 10000 }} />
              </Grid>
            </Grid>
            <FormControlLabel control={<Checkbox color="primary" checked={singleUse} onChange={(event) => setSingleUse(event.target.checked)} />} label={t('singleUse')} />
          </>
        ) : (
          <>
            <Box display="flex" alignItems="center" style={{ gap: 8 }}>
              <CheckCircleIcon color="primary" />
              <Typography variant="h6">{t('inviteCreated')}</Typography>
            </Box>
            <div className={classes.qrBox}>
              <QRCodeSVG value={created.shareURL} size={220} level="M" marginSize={4} title={t('qrTitle')} />
            </div>
            <Typography variant="caption" color="textSecondary">{t('qrHelp')}</Typography>
            <Box mt={1.5} className={classes.shareLink}>{created.shareURL}</Box>
            <Box mt={1.5} display="flex" justifyContent="center" style={{ gap: 8 }}>
              <Button variant="contained" color="primary" startIcon={<LinkIcon />} onClick={() => copyText(created.shareURL).then(onCopied).catch(() => onError({ code: 'copy_failed' }))}>{t('copyLink')}</Button>
              <Button variant="outlined" onClick={() => copyText(created.deepLink).then(onCopied).catch(() => onError({ code: 'copy_failed' }))}>{t('copyDeepLink')}</Button>
            </Box>
          </>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t(created ? 'done' : 'cancel')}</Button>
        {!created && <Button variant="contained" color="primary" onClick={create} disabled={busy || !Number.isInteger(Number(days)) || Number(days) < 1 || Number(days) > 365 || (!singleUse && (!Number.isInteger(Number(maxUses)) || Number(maxUses) < 1 || Number(maxUses) > 10000))}>{busy ? <CircularProgress size={20} aria-label={t('loading')} /> : t('generateInvite')}</Button>}
      </DialogActions>
    </Dialog>
  )
}

function UpdatesCard({ api, locale, t, onBusyChange, onReconnect }) {
  const classes = useStyles()
  const [data, setData] = useState(null)
  const [health, setHealth] = useState(null)
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState('')
  const [error, setError] = useState(null)
  const [loadError, setLoadError] = useState(null)
  const [notice, setNotice] = useState('')
  const [notesOpen, setNotesOpen] = useState(false)
  const [confirmation, setConfirmation] = useState(null)
  const [progress, setProgress] = useState(null)
  const lifetime = useRef(new AbortController())

  useEffect(() => () => lifetime.current.abort(), [])

  const refresh = useCallback(async () => {
    const results = await Promise.allSettled([api.updateStatus(), api.readyz()])
    if (lifetime.current.signal.aborted) return
    if (results[0].status === 'fulfilled') setData(results[0].value)
    setHealth(results[1].status === 'fulfilled' ? results[1].value : null)
    setLoadError(results.find((result) => result.status === 'rejected')?.reason || null)
    setLoading(false)
  }, [api])

  useEffect(() => {
    if (busy) return undefined
    refresh()
    const timer = setInterval(refresh, 30000)
    return () => clearInterval(timer)
  }, [busy, refresh])

  const status = data?.status || {}
  const currentVersion = health?.version || status.currentVersion
  const pluginVersion = health?.pluginVersion
  const pairing = health ? (!pluginVersion ? 'not_paired' : health.plugin) : 'unknown'
  const versionMismatch = status.versionMismatch || health?.versionMismatch || (currentVersion && pluginVersion && !sameVersion(currentVersion, pluginVersion))
  const repository = /^[\w.-]+\/[\w.-]+$/.test(data?.repository || '') ? data.repository : 'mythezone/navidrome-music-room'
  let releaseURL = `https://github.com/${repository}/releases`
  try {
    const url = new URL(status.releaseURL)
    if (url.protocol === 'https:' && url.hostname === 'github.com' && !url.username && !url.password) releaseURL = url.href
  } catch { /* Older gateways may not have release metadata yet. */ }

  const perform = async (action) => {
    setBusy(action)
    onBusyChange(true)
    setError(null)
    setNotice('')
    try {
      if (action === 'check') {
        const result = await api.updateAction('check')
        setData((current) => ({ ...current, ...result }))
      } else if (action === 'diagnostics') {
        await api.exportDiagnostics()
      } else if (action === 'rescan') {
        const result = await api.activateNavidromePlugin()
        if (!result.activated) throw Object.assign(new Error('Plugin activation failed'), { code: result.reason })
        const ready = await waitForReady(api, currentVersion, true, lifetime.current.signal)
        setHealth(ready)
        await onReconnect()
        setNotice('pluginActivated')
      }
    } catch (cause) {
      if (!lifetime.current.signal.aborted) setError(cause)
    } finally {
      if (!lifetime.current.signal.aborted) {
        setBusy('')
        onBusyChange(false)
      }
    }
  }

  const upgrade = async () => {
    const { action, version } = confirmation
    setConfirmation(null)
    setError(null)
    setNotice('')
    setBusy(action)
    onBusyChange(true)
    try {
      const ready = await runUpgrade(api, action, version, (step, target) => {
        if (!lifetime.current.signal.aborted) setProgress({ action, version: target, step })
      }, lifetime.current.signal)
      setHealth(ready)
      await onReconnect()
      setProgress({ action, version: ready.version, step: 4 })
    } catch (cause) {
      if (!lifetime.current.signal.aborted) setProgress((current) => ({ ...current, error: cause }))
    } finally {
      if (!lifetime.current.signal.aborted) {
        setBusy('')
        onBusyChange(false)
      }
    }
  }

  const metrics = [
    ['gatewayVersion', currentVersion || '—'],
    ['pluginVersion', pluginVersion || '—'],
    ['pairing', t(['paired', 'stale', 'not_paired'].includes(pairing) ? pairing : 'unknown')],
    ['channel', t(status.channel === 'beta' ? 'beta' : status.channel === 'stable' ? 'stable' : 'unknown')],
    ['latestVersion', status.latestVersion || '—'],
    ['publishedAt', formatDate(status.releasePublishedAt || status.publishedAt, locale)],
  ]
  const steps = [progress?.action === 'rollback' ? 'stepRollback' : 'stepVerify', 'stepRestart', 'stepReady', 'stepPlugin']

  return (
    <Card className={classes.statusCard}>
      <CardContent>
        <div className={classes.sectionHeader}>
          <SystemUpdateIcon color="primary" />
          <Typography variant="h6">{t('updates')}</Typography>
          {(loading || busy) && <CircularProgress size={20} aria-label={t('loading')} />}
        </div>
        {status.updateAvailable && <Box mb={2} role="status"><Chip color="primary" icon={<SystemUpdateIcon />} label={`${t('updateAvailable')} · ${status.latestVersion || ''}`} /></Box>}
        <Grid container spacing={2}>
          {metrics.map(([label, value]) => <Grid item xs={12} sm={6} md={4} key={label}><div className={classes.metric}><Typography variant="caption" color="textSecondary">{t(label)}</Typography><Typography>{value}</Typography></div></Grid>)}
        </Grid>
        {versionMismatch && <Box mt={2} display="flex" alignItems="center" style={{ gap: 8 }} role="status"><WarningIcon style={{ color: '#ffb74d' }} /><Typography>{t('versionMismatch')}</Typography></Box>}
        {(error || loadError) && <Box mt={2} role="alert"><Typography color="error">{localizeError(locale, error || loadError)}</Typography></Box>}
        {status.lastError && <Box mt={2}><Typography color="error">{t('updateFailed')}</Typography></Box>}
        {notice && <Box mt={2} role="status"><Typography>{t(notice)}</Typography></Box>}
        <div className={classes.detailActions}>
          <Button variant="outlined" startIcon={<RefreshIcon />} disabled={Boolean(busy)} onClick={() => perform('check')}>{t('checkUpdates')}</Button>
          <Button variant="outlined" disabled={Boolean(busy)} onClick={() => setNotesOpen(true)}>{t('releaseNotes')}</Button>
          <Button href={releaseURL} target="_blank" rel="noopener noreferrer">{t('releasePage')}</Button>
          {data?.managedByLauncher && status.updateAvailable && <Button variant="contained" color="primary" startIcon={<SystemUpdateIcon />} disabled={Boolean(busy) || !status.latestVersion} onClick={() => setConfirmation({ action: 'install', version: status.latestVersion })}>{t('upgradeNow')}</Button>}
          {(versionMismatch || pairing === 'stale' || pairing === 'not_paired') && <Button variant="outlined" startIcon={<RefreshIcon />} disabled={Boolean(busy) || !currentVersion} onClick={() => perform('rescan')}>{t('rescanPlugin')}</Button>}
          {status.rollbackAvailable && <Button variant="outlined" startIcon={<HistoryIcon />} disabled={Boolean(busy) || !data?.managedByLauncher} onClick={() => setConfirmation({ action: 'rollback', version: status.rollbackVersion || t('unknown') })}>{t('rollback')}</Button>}
          <Button variant="outlined" startIcon={<DownloadIcon />} disabled={Boolean(busy)} onClick={() => perform('diagnostics')}>{t('diagnostics')}</Button>
        </div>
        {data?.managedByLauncher === false && <Box mt={2}>
          <Typography variant="body2" color="textSecondary">{t('manualUpgrade')}</Typography>
          <Box component="pre" className={classes.shareLink}>docker compose pull &amp;&amp; docker compose up -d</Box>
          <Button href={releaseURL} target="_blank" rel="noopener noreferrer">{t('releasePage')}</Button>
        </Box>}
      </CardContent>
      <Dialog open={notesOpen} onClose={() => setNotesOpen(false)} fullWidth maxWidth="md" aria-labelledby="release-notes-title">
        <DialogTitle id="release-notes-title">{t('releaseNotes')}</DialogTitle>
        <DialogContent dividers><Typography component="pre" style={{ whiteSpace: 'pre-wrap', overflowWrap: 'anywhere', fontFamily: 'inherit' }}>{status.releaseNotes || t('noReleaseNotes')}</Typography></DialogContent>
        <DialogActions><Button href={releaseURL} target="_blank" rel="noopener noreferrer">{t('releasePage')}</Button><Button onClick={() => setNotesOpen(false)}>{t('close')}</Button></DialogActions>
      </Dialog>
      <Dialog open={Boolean(confirmation)} onClose={() => setConfirmation(null)} fullWidth maxWidth="sm" aria-labelledby="update-confirm-title" aria-describedby="update-confirm-body">
        <DialogTitle id="update-confirm-title">{t(confirmation?.action === 'rollback' ? 'rollback' : 'upgradeNow')}</DialogTitle>
        <DialogContent><Typography id="update-confirm-body">{t(confirmation?.action === 'rollback' ? 'rollbackConfirm' : 'upgradeConfirm', { version: confirmation?.version })}</Typography></DialogContent>
        <DialogActions><Button onClick={() => setConfirmation(null)}>{t('cancel')}</Button><Button variant="contained" color="primary" onClick={upgrade}>{t(confirmation?.action === 'rollback' ? 'rollback' : 'upgradeNow')}</Button></DialogActions>
      </Dialog>
      <Dialog open={Boolean(progress)} onClose={busy ? undefined : () => setProgress(null)} fullWidth maxWidth="sm" aria-labelledby="upgrade-progress-title">
        <DialogTitle id="upgrade-progress-title">{t(progress?.action === 'rollback' ? 'rollback' : 'upgradeNow')}</DialogTitle>
        <DialogContent dividers>
          <div aria-live="polite" aria-atomic="true">
            <Stepper orientation="vertical" activeStep={progress?.step || 0} style={{ background: 'transparent', padding: 0 }}>
              {steps.map((key, index) => <Step key={key} completed={index < (progress?.step || 0)}><StepLabel error={Boolean(progress?.error) && index === progress?.step}>{t(key, { version: progress?.version })}</StepLabel></Step>)}
            </Stepper>
            {busy && <Box mt={2}><CircularProgress size={24} aria-label={t('loading')} /></Box>}
            {progress?.step === 4 && <Box mt={2} display="flex" style={{ gap: 8 }}><CheckCircleIcon color="primary" /><Typography>{t('upgradeComplete', { version: progress.version })}</Typography></Box>}
          </div>
          {progress?.error && <Box mt={2} role="alert"><Typography color="error">{localizeError(locale, progress.error)}</Typography></Box>}
        </DialogContent>
        <DialogActions><Button onClick={() => setProgress(null)} disabled={Boolean(busy)}>{t(progress?.step === 4 ? 'done' : 'close')}</Button></DialogActions>
      </Dialog>
    </Card>
  )
}

function App() {
  const classes = useStyles()
  const [locale, setLocale] = useState(() => detectLocale())
  const t = useMemo(() => (key, params) => translate(locale, key, params), [locale])
  useEffect(() => {
    document.title = t('pageTitle')
    document.documentElement.lang = locale
    try { localStorage.setItem('nmr:locale', locale) } catch { /* Keep the selected language for this page when storage is blocked. */ }
  }, [locale, t])
  const api = useMemo(() => new GatewayAPI(), [])
  const [proof] = useState(() => readNavidromeProof())
  const [auth, setAuth] = useState({ state: 'loading', session: null, error: null })
  const [rooms, setRooms] = useState([])
  const [selectedID, setSelectedID] = useState('')
  const [folders, setFolders] = useState([])
  const [members, setMembers] = useState([])
  const [invites, setInvites] = useState([])
  const [detailsLoading, setDetailsLoading] = useState(false)
  const [roomDialog, setRoomDialog] = useState(null)
  const [shareOpen, setShareOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [notice, setNotice] = useState('')
  const [error, setError] = useState(null)

  const selectedRoom = rooms.find((room) => room.roomID === selectedID) || null

  const showError = useCallback((cause) => {
    setError(cause)
  }, [])

  const refreshRooms = useCallback(async (preferredID) => {
    const payload = await api.rooms()
    const nextRooms = Array.isArray(payload.rooms) ? payload.rooms : []
    setRooms(nextRooms)
    setSelectedID((current) => {
      const wanted = preferredID || current
      if (wanted && nextRooms.some((room) => room.roomID === wanted)) return wanted
      return nextRooms[0]?.roomID || ''
    })
  }, [api])

  useEffect(() => {
    let active = true
    async function boot() {
      if (!hasCompleteProof(proof)) {
        setAuth({ state: 'missing', session: null, error: null })
        return
      }
      try {
        const session = await api.exchange(proof)
        if (!active) return
        setAuth({ state: 'ready', session, error: null })
        const [folderList] = await Promise.all([
          api.musicFolders(proof).catch(() => session.user.musicFolderIDs.map((id) => ({ id, name: '' }))),
          refreshRooms(),
        ])
        if (active) setFolders(folderList.map((folder) => ({ id: Number(folder.id), name: folder.name || '' })))
      } catch (cause) {
        if (active) setAuth({ state: 'error', session: null, error: cause })
      }
    }
    boot()
    return () => { active = false }
  }, [api, proof, refreshRooms])

  useEffect(() => {
    if (auth.state !== 'ready' || busy) return undefined
    const timer = window.setInterval(() => refreshRooms().catch(showError), 15000)
    return () => window.clearInterval(timer)
  }, [auth.state, busy, refreshRooms, showError])

  const refreshDetails = useCallback(async () => {
    if (!selectedID) {
      setMembers([])
      setInvites([])
      return
    }
    setDetailsLoading(true)
    try {
      const inviteRequest = auth.session?.user.adminRole ? api.invites(selectedID) : Promise.resolve({ invites: [] })
      const [memberPayload, invitePayload] = await Promise.all([api.members(selectedID), inviteRequest])
      setMembers(Array.isArray(memberPayload.members) ? memberPayload.members : [])
      setInvites(Array.isArray(invitePayload.invites) ? invitePayload.invites : [])
    } catch (cause) {
      showError(cause)
    } finally {
      setDetailsLoading(false)
    }
  }, [api, auth.session?.user.adminRole, selectedID, showError])

  useEffect(() => { refreshDetails() }, [refreshDetails])

  const renewSession = async () => {
    const session = await api.exchange(proof)
    setAuth({ state: 'ready', session, error: null })
    await refreshRooms()
    await refreshDetails()
  }

  const saveRoom = async (values) => {
    setBusy(true)
    try {
      const room = roomDialog === 'edit' ? await api.updateRoom(selectedID, values) : await api.createRoom(values)
      setRoomDialog(null)
      setNotice(roomDialog === 'edit' ? 'roomSaved' : 'roomCreated')
      await refreshRooms(room.roomID)
    } catch (cause) { showError(cause) } finally { setBusy(false) }
  }

  const toggleRoom = async () => {
    if (!selectedRoom) return
    setBusy(true)
    try {
      await api.setRoomOpen(selectedRoom.roomID, selectedRoom.status !== 'open')
      setNotice(selectedRoom.status === 'open' ? 'roomClosed' : 'roomReopened')
      await refreshRooms(selectedRoom.roomID)
    } catch (cause) { showError(cause) } finally { setBusy(false) }
  }

  const deleteRoom = async () => {
    if (!selectedRoom || !window.confirm(t('deleteConfirm', { name: selectedRoom.name }))) return
    setBusy(true)
    try {
      await api.deleteRoom(selectedRoom.roomID)
      setNotice('roomDeleted')
      await refreshRooms()
    } catch (cause) { showError(cause) } finally { setBusy(false) }
  }

  const createInvite = async (values) => {
    setBusy(true)
    try {
      const invite = await api.createInvite(selectedRoom.roomID, values)
      await refreshDetails()
      return invite
    } catch (cause) {
      showError(cause)
      return null
    } finally { setBusy(false) }
  }

  const revokeInvite = async (inviteID) => {
    setBusy(true)
    try {
      await api.revokeInvite(selectedRoom.roomID, inviteID)
      setNotice('inviteRevoked')
      await refreshDetails()
    } catch (cause) { showError(cause) } finally { setBusy(false) }
  }

  const removeMember = async (member) => {
    if (!window.confirm(t('removeConfirm', { name: member.displayName || member.username }))) return
    setBusy(true)
    try {
      await api.removeMember(selectedRoom.roomID, member.username)
      setNotice('memberRemoved')
      await refreshDetails()
    } catch (cause) { showError(cause) } finally { setBusy(false) }
  }

  const header = (
    <AppBar position="static" className={classes.appBar}>
      <Toolbar className={classes.toolbar}>
        <Tooltip title={t('back')}><IconButton color="inherit" href="/app/" aria-label={t('back')}><ArrowBackIcon /></IconButton></Tooltip>
        <div className={classes.brand}><HeadsetIcon color="primary" /><Typography className={classes.brandTitle} variant="h6">{t('appName')}</Typography><Chip className={classes.desktopOnly} size="small" label={t('pluginLabel')} variant="outlined" /></div>
        {auth.session && <Typography className={classes.desktopOnly} variant="body2">{auth.session.user.displayName || auth.session.user.username}</Typography>}
        <LanguageIcon className={classes.desktopOnly} />
        <Select native value={locale} onChange={(event) => setLocale(event.target.value)} inputProps={{ 'aria-label': t('language') }} style={{ maxWidth: 135, color: 'inherit' }}>
          {Object.entries(locales).map(([value, label]) => <option key={value} value={value}>{label}</option>)}
        </Select>
        {auth.state === 'ready' && <Tooltip title={t('refresh')}><IconButton color="inherit" disabled={busy} aria-label={t('refresh')} onClick={() => refreshRooms().catch(showError)}><RefreshIcon /></IconButton></Tooltip>}
      </Toolbar>
    </AppBar>
  )

  if (auth.state === 'loading') {
    return <div className={classes.root}>{header}<div className={classes.loading}><CircularProgress aria-label={t('loading')} /></div></div>
  }

  if (auth.state !== 'ready') {
    return (
      <div className={classes.root}>
        {header}
        <Paper className={classes.loginCard}>
          <LockIcon color="primary" style={{ fontSize: 48 }} />
          <Typography variant="h5" gutterBottom>{t('loginTitle')}</Typography>
          <Typography color="textSecondary" paragraph>
            {auth.state === 'missing' ? t('loginHelp') : t('sessionFailed', { error: localizeError(locale, auth.error) })}
          </Typography>
          <Button color="primary" variant="contained" href="/app/" startIcon={<ArrowBackIcon />}>{t('back')}</Button>
        </Paper>
      </div>
    )
  }

  const isAdmin = auth.session.user.adminRole
  const activeInvites = invites.filter((invite) => !invite.revokedAt && new Date(invite.expiresAt) > new Date() && invite.useCount < invite.maxUses)

  return (
    <div className={classes.root}>
      {header}

      <main className={classes.content}>
        <Card className={classes.statusCard}>
          <CardContent className={classes.statusRow}>
            <CheckCircleIcon color="primary" />
            <Typography variant="subtitle1">{t('connected')}</Typography>
            <Chip className={classes.desktopOnly} size="small" color="primary" variant="outlined" label={t('gatewayAddress', { url: auth.session.gatewayBaseURL })} />
            <Chip size="small" variant="outlined" label={t('openRooms', { count: rooms.filter((room) => room.status === 'open').length })} />
            <div className={classes.grow} />
            <Typography variant="caption" color="textSecondary">{t('mediaHelp')}</Typography>
          </CardContent>
        </Card>

        {isAdmin && <UpdatesCard api={api} locale={locale} t={t} onBusyChange={setBusy} onReconnect={renewSession} />}

        {!isAdmin && <Box mb={2}><Card><CardContent><Typography color="textSecondary">{t('notAdmin')}</Typography></CardContent></Card></Box>}

        <Grid container spacing={2}>
          <Grid item xs={12} md={4}>
            <Paper className={classes.panel}>
              <div className={classes.panelHeader}>
                <div><Typography variant="h6">{t('rooms')}</Typography><Typography variant="caption" color="textSecondary">{t('visibleRooms', { count: rooms.length })}</Typography></div>
                <div className={classes.grow} />
                {isAdmin && <Button variant="contained" color="primary" disabled={busy} startIcon={<AddIcon />} onClick={() => setRoomDialog('create')}>{t('create')}</Button>}
              </div>
              <Divider />
              {rooms.length ? (
                <List disablePadding>
                  {rooms.map((room) => (
                    <ListItem key={room.roomID} button selected={room.roomID === selectedID} onClick={() => setSelectedID(room.roomID)} className={`${classes.roomItem} ${room.roomID === selectedID ? classes.selectedRoom : ''}`}>
                      <ListItemIcon><div className={classes.roomAvatar}><RoomIcon /></div></ListItemIcon>
                      <ListItemText primary={room.name} secondary={`${t('onlineCount', { count: room.onlineCount })} · ${t(room.playbackMode)}`} />
                      <Chip size="small" label={t(room.status === 'open' ? 'open' : 'closed')} color={room.status === 'open' ? 'primary' : 'default'} variant={room.status === 'open' ? 'outlined' : 'default'} />
                    </ListItem>
                  ))}
                </List>
              ) : (
                <div className={classes.empty}><div><HeadsetIcon style={{ fontSize: 54, opacity: .5 }} /><Typography variant="h6">{t('noRooms')}</Typography><Typography variant="body2">{t('noRoomsHelp')}</Typography></div></div>
              )}
            </Paper>
          </Grid>

          <Grid item xs={12} md={8}>
            <Paper className={classes.panel}>
              {!selectedRoom ? (
                <div className={classes.empty}><div><RoomIcon style={{ fontSize: 54, opacity: .5 }} /><Typography variant="h6">{t('selectRoom')}</Typography><Typography variant="body2">{t('selectRoomHelp')}</Typography></div></div>
              ) : (
                <>
                  <div className={classes.panelHeader}>
                    <div><Box display="flex" alignItems="center" style={{ gap: 8 }}><Typography variant="h5">{selectedRoom.name}</Typography><Chip size="small" color={selectedRoom.status === 'open' ? 'primary' : 'default'} label={t(selectedRoom.status === 'open' ? 'open' : 'closed')} /></Box><Typography variant="caption" color="textSecondary">{t('roomID', { id: selectedRoom.roomID })}</Typography></div>
                    <div className={classes.grow} />
                    {isAdmin && <Tooltip title={t('roomSettings')}><IconButton disabled={busy} aria-label={t('roomSettings')} onClick={() => setRoomDialog('edit')}><EditIcon /></IconButton></Tooltip>}
                  </div>
                  <Divider />
                  <div className={classes.detailBody}>
                    <Grid container spacing={2}>
                      <Grid item xs={6} sm={3}><div className={classes.metric}><Typography variant="caption" color="textSecondary">{t('online')}</Typography><Typography variant="h6">{selectedRoom.onlineCount}</Typography></div></Grid>
                      <Grid item xs={6} sm={3}><div className={classes.metric}><Typography variant="caption" color="textSecondary">{t('members')}</Typography><Typography variant="h6">{members.filter((member) => member.active).length}</Typography></div></Grid>
                      <Grid item xs={6} sm={3}><div className={classes.metric}><Typography variant="caption" color="textSecondary">{t('activeInvites')}</Typography><Typography variant="h6">{activeInvites.length}</Typography></div></Grid>
                      <Grid item xs={6} sm={3}><div className={classes.metric}><Typography variant="caption" color="textSecondary">{t('queueMode')}</Typography><Typography variant="h6">{t(selectedRoom.playbackMode)}</Typography></div></Grid>
                      <Grid item xs={12}><div className={classes.metric}><Typography variant="caption" color="textSecondary">{t('skipPolicy')}</Typography><Typography>{t((selectedRoom.allowMemberSkip ?? true) ? 'allMembers' : 'managersOnly')}</Typography></div></Grid>
                    </Grid>

                    {isAdmin && <div className={classes.detailActions}>
                      <Button variant="outlined" startIcon={<WebIcon />} disabled={selectedRoom.status !== 'open'} href={`../join/${selectedRoom.roomID}/`}>{t('enterRoom')}</Button>
                      <Button variant="contained" color="primary" startIcon={<ShareIcon />} disabled={busy || selectedRoom.status !== 'open'} onClick={() => setShareOpen(true)}>{t('share')}</Button>
                      <Button variant="outlined" startIcon={<PowerIcon />} disabled={busy} onClick={toggleRoom}>{t(selectedRoom.status === 'open' ? 'closeRoom' : 'reopen')}</Button>
                      <Button variant="outlined" className={classes.danger} startIcon={<DeleteIcon />} disabled={busy} onClick={deleteRoom}>{t('delete')}</Button>
                    </div>}

                    <section className={classes.section}>
                      <div className={classes.sectionHeader}><GroupIcon color="primary" /><Typography variant="h6">{t('members')}</Typography>{detailsLoading && <CircularProgress size={16} aria-label={t('loading')} />}</div>
                      <Divider />
                      <List dense>
                        {members.map((member) => (
                          <ListItem key={member.username}>
                            <ListItemIcon><div className={member.online ? classes.onlineDot : classes.offlineDot} /></ListItemIcon>
                            <ListItemText primary={member.displayName || member.username} secondary={`@${member.username} · ${t(member.role === 'owner' ? 'owner' : 'member')} · ${member.online ? t('online') : t('lastSeen', { date: formatDate(member.lastSeenAt, locale) })}`} />
                            {isAdmin && member.role !== 'owner' && <ListItemSecondaryAction><Tooltip title={t('removeMember')}><IconButton edge="end" aria-label={t('removeMember')} onClick={() => removeMember(member)} disabled={busy}><DeleteIcon /></IconButton></Tooltip></ListItemSecondaryAction>}
                          </ListItem>
                        ))}
                      </List>
                    </section>

                    <section className={classes.section}>
                      <div className={classes.sectionHeader}><LinkIcon color="primary" /><Typography variant="h6">{t('inviteHistory')}</Typography></div>
                      <Divider />
                      <List dense>
                        {invites.length ? invites.map((invite) => {
                          const active = !invite.revokedAt && new Date(invite.expiresAt) > new Date() && invite.useCount < invite.maxUses
                          return <ListItem key={invite.inviteID}><ListItemText primary={invite.label || t('unnamedInvite')} secondary={`${t('inviteUsage', { count: invite.useCount, max: invite.maxUses, date: formatDate(invite.expiresAt, locale) })}${invite.revokedAt ? ` · ${t('revoked')}` : ''}`} /><ListItemSecondaryAction>{active && isAdmin ? <Button size="small" onClick={() => revokeInvite(invite.inviteID)} disabled={busy}>{t('revoke')}</Button> : <Chip size="small" label={t(active ? 'active' : 'inactive')} variant="outlined" />}</ListItemSecondaryAction></ListItem>
                        }) : <ListItem><ListItemText primary={t('noInvites')} secondary={t('noInvitesHelp')} /></ListItem>}
                      </List>
                    </section>

                    <section className={`${classes.section} ${classes.roadmap}`}>
                      <div className={classes.sectionHeader}><CodeIcon /><Typography variant="subtitle1">{t('roadmap')}</Typography></div>
                      <Typography variant="body2" color="textSecondary">{t('roadmapHelp')}</Typography>
                    </section>
                  </div>
                </>
              )}
            </Paper>
          </Grid>
        </Grid>
      </main>

      <RoomDialog t={t} open={Boolean(roomDialog)} room={roomDialog === 'edit' ? selectedRoom : null} folders={folders} busy={busy} onClose={() => setRoomDialog(null)} onSave={saveRoom} />
      <ShareDialog t={t} onError={showError} open={shareOpen} room={selectedRoom} busy={busy} onClose={() => setShareOpen(false)} onCreate={createInvite} onCopied={() => setNotice('copied')} />
      <Snackbar className={classes.snackbar} open={Boolean(notice)} autoHideDuration={3500} onClose={() => setNotice('')} message={notice ? t(notice) : ''} action={<IconButton size="small" color="inherit" aria-label={t('close')} onClick={() => setNotice('')}><CloseIcon fontSize="small" /></IconButton>} />
      <Snackbar className={classes.snackbar} open={Boolean(error)} autoHideDuration={7000} onClose={() => setError(null)} message={error ? localizeError(locale, error) : ''} action={<IconButton size="small" color="inherit" aria-label={t('close')} onClick={() => setError(null)}><CloseIcon fontSize="small" /></IconButton>} />
    </div>
  )
}

export class ErrorBoundary extends React.Component {
  state = { failed: false }

  static getDerivedStateFromError() { return { failed: true } }

  componentDidCatch() {
    const locale = detectLocale()
    document.documentElement.lang = locale
    document.title = translate(locale, 'pageTitle')
  }

  render() {
    if (!this.state.failed) return this.props.children
    const t = (key) => translate(detectLocale(), key)
    return (
      <main role="alert" style={{ maxWidth: 560, margin: '12vh auto', padding: 32, borderRadius: 12, background: '#303030', color: '#fff', fontFamily: 'system-ui, sans-serif', textAlign: 'center' }}>
        <h1 style={{ fontSize: 24 }}>{t('recoveryTitle')}</h1>
        <p>{t('recoveryBody')}</p>
        <button type="button" onClick={() => window.location.reload()} style={{ padding: '10px 18px', margin: 8, cursor: 'pointer' }}>{t('reload')}</button>
        <a href="/app/" style={{ display: 'inline-block', padding: 10, color: '#90caf9' }}>{t('back')}</a>
      </main>
    )
  }
}

export default App
