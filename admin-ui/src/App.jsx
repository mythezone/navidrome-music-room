import React, { useCallback, useEffect, useMemo, useState } from 'react'
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
  MeetingRoom as RoomIcon,
  PowerSettingsNew as PowerIcon,
  Refresh as RefreshIcon,
  Share as ShareIcon,
  Web as WebIcon,
} from '@material-ui/icons'
import { QRCodeSVG } from 'qrcode.react'
import { GatewayAPI, hasCompleteProof, readNavidromeProof } from './api'

const useStyles = makeStyles((theme) => ({
  root: { minHeight: '100vh', background: 'linear-gradient(160deg, #303030 0%, #263238 100%)' },
  appBar: { background: '#212121', color: '#fff', boxShadow: '0 2px 10px rgba(0,0,0,.35)' },
  toolbar: { minHeight: 64, gap: theme.spacing(1.5), [theme.breakpoints.down('xs')]: { gap: theme.spacing(.5), paddingLeft: theme.spacing(1), paddingRight: theme.spacing(1) } },
  brand: { display: 'flex', alignItems: 'center', gap: theme.spacing(1), flexGrow: 1, minWidth: 0 },
  brandTitle: { whiteSpace: 'nowrap' },
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
  metric: { padding: theme.spacing(1.5), background: 'rgba(0,0,0,.12)', borderRadius: 8 },
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

function formatDate(value) {
  if (!value) return '—'
  return new Intl.DateTimeFormat('zh-CN', { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value))
}

function copyText(value) {
  if (navigator.clipboard && window.isSecureContext) return navigator.clipboard.writeText(value)
  const node = document.createElement('textarea')
  node.value = value
  node.style.position = 'fixed'
  node.style.opacity = '0'
  document.body.appendChild(node)
  node.select()
  document.execCommand('copy')
  node.remove()
  return Promise.resolve()
}

function RoomDialog({ open, room, folders, busy, onClose, onSave }) {
  const [name, setName] = useState('')
  const [queueLimit, setQueueLimit] = useState(20)
  const [playbackMode, setPlaybackMode] = useState('fifo')
  const [preload, setPreload] = useState(true)
  const [selectedFolders, setSelectedFolders] = useState([])

  useEffect(() => {
    setName(room?.name || '')
    setQueueLimit(room?.queueLimit || 20)
    setPlaybackMode(room?.playbackMode || 'fifo')
    setPreload(room?.preloadNextTrack ?? true)
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
    })
  }

  return (
    <Dialog open={open} onClose={busy ? undefined : onClose} fullWidth maxWidth="sm" aria-labelledby="room-dialog-title">
      <form onSubmit={submit}>
        <DialogTitle id="room-dialog-title">{room ? '房间设置' : '创建听歌房'}</DialogTitle>
        <DialogContent dividers>
          <TextField id="room-name" label="房间名称" value={name} onChange={(event) => setName(event.target.value)} fullWidth required inputProps={{ maxLength: 80 }} margin="normal" autoFocus />
          <Grid container spacing={2}>
            <Grid item xs={12} sm={6}>
              <TextField id="room-queue-limit" label="每人待播上限" type="number" value={queueLimit} onChange={(event) => setQueueLimit(event.target.value)} fullWidth margin="normal" inputProps={{ min: 1, max: 100 }} />
            </Grid>
            <Grid item xs={12} sm={6}>
              <FormControl fullWidth margin="normal">
                <InputLabel id="playback-mode-label">队列模式</InputLabel>
                <Select labelId="playback-mode-label" value={playbackMode} onChange={(event) => setPlaybackMode(event.target.value)}>
                  <MenuItem value="fifo">按点歌顺序</MenuItem>
                  <MenuItem value="fair_random">公平随机</MenuItem>
                </Select>
              </FormControl>
            </Grid>
          </Grid>
          <Typography variant="subtitle2">可使用的音乐库</Typography>
          <FormGroup>
            {folders.map((folder) => (
              <FormControlLabel
                key={folder.id}
                control={<Checkbox color="primary" checked={selectedFolders.includes(folder.id)} onChange={(event) => setSelectedFolders((current) => event.target.checked ? [...current, folder.id] : current.filter((id) => id !== folder.id))} />}
                label={folder.name}
              />
            ))}
          </FormGroup>
          {!folders.length && <Typography variant="body2" color="textSecondary">当前账号没有可用的 Navidrome 音乐库。</Typography>}
          <FormControlLabel control={<Checkbox color="primary" checked={preload} onChange={(event) => setPreload(event.target.checked)} />} label="预加载下一首" />
        </DialogContent>
        <DialogActions>
          <Button onClick={onClose} disabled={busy}>取消</Button>
          <Button color="primary" variant="contained" type="submit" disabled={busy || !name.trim() || !selectedFolders.length}>
            {busy ? <CircularProgress size={20} /> : room ? '保存' : '创建房间'}
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  )
}

function ShareDialog({ open, room, busy, onClose, onCreate, onCopied }) {
  const classes = useStyles()
  const [label, setLabel] = useState('分享邀请')
  const [days, setDays] = useState(7)
  const [maxUses, setMaxUses] = useState(20)
  const [singleUse, setSingleUse] = useState(false)
  const [created, setCreated] = useState(null)

  useEffect(() => {
    if (open) {
      setCreated(null)
      setLabel('分享邀请')
      setDays(7)
      setMaxUses(20)
      setSingleUse(false)
    }
  }, [open, room?.roomID])

  const create = async () => {
    const expiresAt = new Date(Date.now() + Number(days) * 86400000).toISOString()
    const invite = await onCreate({ label, expiresAt, maxUses: Number(maxUses), singleUse })
    if (invite) setCreated(invite)
  }

  return (
    <Dialog open={open} onClose={busy ? undefined : onClose} fullWidth maxWidth="sm" aria-labelledby="share-dialog-title">
      <DialogTitle id="share-dialog-title">分享「{room?.name || ''}」</DialogTitle>
      <DialogContent dividers>
        {!created ? (
          <>
            <Typography color="textSecondary" gutterBottom>创建一条可撤销的邀请。密钥只显示这一次，二维码在当前浏览器本地生成。</Typography>
            <TextField id="invite-label" label="邀请备注" value={label} onChange={(event) => setLabel(event.target.value)} fullWidth margin="normal" inputProps={{ maxLength: 80 }} />
            <Grid container spacing={2}>
              <Grid item xs={6}>
                <TextField id="invite-days" label="有效天数" type="number" value={days} onChange={(event) => setDays(event.target.value)} fullWidth margin="normal" inputProps={{ min: 1, max: 365 }} />
              </Grid>
              <Grid item xs={6}>
                <TextField id="invite-max-uses" label="最多使用次数" type="number" value={singleUse ? 1 : maxUses} onChange={(event) => setMaxUses(event.target.value)} fullWidth margin="normal" disabled={singleUse} inputProps={{ min: 1, max: 10000 }} />
              </Grid>
            </Grid>
            <FormControlLabel control={<Checkbox color="primary" checked={singleUse} onChange={(event) => setSingleUse(event.target.checked)} />} label="仅允许使用一次" />
          </>
        ) : (
          <>
            <Box display="flex" alignItems="center" style={{ gap: 8 }}>
              <CheckCircleIcon color="primary" />
              <Typography variant="h6">邀请已生成</Typography>
            </Box>
            <div className={classes.qrBox}>
              <QRCodeSVG value={created.shareURL} size={220} level="M" includeMargin />
            </div>
            <Typography variant="caption" color="textSecondary">扫码或打开链接可直接进入 Web 听歌房，也可以交给 MusicMate 处理。邀请密钥位于 URL fragment，不会进入服务器访问日志。</Typography>
            <Box mt={1.5} className={classes.shareLink}>{created.shareURL}</Box>
            <Box mt={1.5} display="flex" justifyContent="center" style={{ gap: 8 }}>
              <Button variant="contained" color="primary" startIcon={<LinkIcon />} onClick={() => copyText(created.shareURL).then(onCopied)}>复制房间链接</Button>
              <Button variant="outlined" onClick={() => copyText(created.deepLink).then(onCopied)}>复制 Deep Link</Button>
            </Box>
          </>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{created ? '完成' : '取消'}</Button>
        {!created && <Button variant="contained" color="primary" onClick={create} disabled={busy || Number(days) < 1 || Number(maxUses) < 1}>{busy ? <CircularProgress size={20} /> : '生成链接和二维码'}</Button>}
      </DialogActions>
    </Dialog>
  )
}

function App() {
  const classes = useStyles()
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
  const [error, setError] = useState('')

  const selectedRoom = rooms.find((room) => room.roomID === selectedID) || null

  const showError = useCallback((cause) => {
    setError(`${cause.message || cause}${cause.code ? ` (${cause.code})` : ''}`)
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
          api.musicFolders(proof).catch(() => session.user.musicFolderIDs.map((id) => ({ id, name: `音乐库 ${id}` }))),
          refreshRooms(),
        ])
        if (active) setFolders(folderList.map((folder) => ({ id: Number(folder.id), name: folder.name || `音乐库 ${folder.id}` })))
      } catch (cause) {
        if (active) setAuth({ state: 'error', session: null, error: cause })
      }
    }
    boot()
    return () => { active = false }
  }, [api, proof, refreshRooms])

  useEffect(() => {
    if (auth.state !== 'ready') return undefined
    const timer = window.setInterval(() => refreshRooms().catch(showError), 15000)
    return () => window.clearInterval(timer)
  }, [auth.state, refreshRooms, showError])

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

  const saveRoom = async (values) => {
    setBusy(true)
    try {
      const room = roomDialog === 'edit' ? await api.updateRoom(selectedID, values) : await api.createRoom(values)
      setRoomDialog(null)
      setNotice(roomDialog === 'edit' ? '房间设置已保存' : '听歌房已创建')
      await refreshRooms(room.roomID)
    } catch (cause) { showError(cause) } finally { setBusy(false) }
  }

  const toggleRoom = async () => {
    if (!selectedRoom) return
    setBusy(true)
    try {
      await api.setRoomOpen(selectedRoom.roomID, selectedRoom.status !== 'open')
      setNotice(selectedRoom.status === 'open' ? '房间已关闭' : '房间已重新开启')
      await refreshRooms(selectedRoom.roomID)
    } catch (cause) { showError(cause) } finally { setBusy(false) }
  }

  const deleteRoom = async () => {
    if (!selectedRoom || !window.confirm(`确定永久删除「${selectedRoom.name}」吗？房间成员、队列和历史也会被删除。`)) return
    setBusy(true)
    try {
      await api.deleteRoom(selectedRoom.roomID)
      setNotice('房间已删除')
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
      setNotice('邀请已撤销；已经加入的成员不受影响')
      await refreshDetails()
    } catch (cause) { showError(cause) } finally { setBusy(false) }
  }

  const removeMember = async (member) => {
    if (!window.confirm(`从房间移除 ${member.displayName || member.username}？`)) return
    setBusy(true)
    try {
      await api.removeMember(selectedRoom.roomID, member.username)
      setNotice('成员已移除')
      await refreshDetails()
    } catch (cause) { showError(cause) } finally { setBusy(false) }
  }

  if (auth.state === 'loading') {
    return <div className={classes.root}><div className={classes.loading}><CircularProgress /></div></div>
  }

  if (auth.state !== 'ready') {
    return (
      <div className={classes.root}>
        <AppBar position="static" className={classes.appBar}><Toolbar><HeadsetIcon /><Typography variant="h6">听歌房</Typography></Toolbar></AppBar>
        <Paper className={classes.loginCard}>
          <LockIcon color="primary" style={{ fontSize: 48 }} />
          <Typography variant="h5" gutterBottom>请先登录 Navidrome</Typography>
          <Typography color="textSecondary" paragraph>
            {auth.state === 'missing' ? '管理页会安全复用当前 Navidrome 登录的 OpenSubsonic 凭据。请登录后，从“设置 → 插件 → Navidrome Music Room → Website”重新打开。' : `无法建立听歌房会话：${auth.error?.message || '未知错误'}`}
          </Typography>
          <Button color="primary" variant="contained" href="/app/" startIcon={<ArrowBackIcon />}>返回 Navidrome 登录</Button>
        </Paper>
      </div>
    )
  }

  const isAdmin = auth.session.user.adminRole
  const activeInvites = invites.filter((invite) => !invite.revokedAt && new Date(invite.expiresAt) > new Date() && invite.useCount < invite.maxUses)

  return (
    <div className={classes.root}>
      <AppBar position="static" className={classes.appBar}>
        <Toolbar className={classes.toolbar}>
          <Tooltip title="返回 Navidrome"><IconButton color="inherit" href="/app/"><ArrowBackIcon /></IconButton></Tooltip>
          <div className={classes.brand}><HeadsetIcon color="primary" /><Typography className={classes.brandTitle} variant="h6">听歌房</Typography><Chip className={classes.desktopOnly} size="small" label="Navidrome Plugin" variant="outlined" /></div>
          <Typography className={classes.desktopOnly} variant="body2">{auth.session.user.displayName || auth.session.user.username}</Typography>
          <Tooltip title="刷新"><IconButton color="inherit" onClick={() => refreshRooms().catch(showError)}><RefreshIcon /></IconButton></Tooltip>
        </Toolbar>
      </AppBar>

      <main className={classes.content}>
        <Card className={classes.statusCard}>
          <CardContent className={classes.statusRow}>
            <CheckCircleIcon color="primary" />
            <Typography variant="subtitle1">房间服务已连接</Typography>
            <Chip className={classes.desktopOnly} size="small" color="primary" variant="outlined" label={`网关 ${auth.session.gatewayBaseURL}`} />
            <Chip size="small" variant="outlined" label={`${rooms.filter((room) => room.status === 'open').length} 个房间开启中`} />
            <div className={classes.grow} />
            <Typography variant="caption" color="textSecondary">媒体仍由各成员账号直接从 Navidrome 播放</Typography>
          </CardContent>
        </Card>

        {!isAdmin && <Box mb={2}><Card><CardContent><Typography color="textSecondary">当前账号不是 Navidrome 管理员，可以查看已加入的房间，但不能创建或管理房间。</Typography></CardContent></Card></Box>}

        <Grid container spacing={2}>
          <Grid item xs={12} md={4}>
            <Paper className={classes.panel}>
              <div className={classes.panelHeader}>
                <div><Typography variant="h6">房间</Typography><Typography variant="caption" color="textSecondary">{rooms.length} 个可见房间</Typography></div>
                <div className={classes.grow} />
                {isAdmin && <Button variant="contained" color="primary" startIcon={<AddIcon />} onClick={() => setRoomDialog('create')}>创建</Button>}
              </div>
              <Divider />
              {rooms.length ? (
                <List disablePadding>
                  {rooms.map((room) => (
                    <ListItem key={room.roomID} button selected={room.roomID === selectedID} onClick={() => setSelectedID(room.roomID)} className={`${classes.roomItem} ${room.roomID === selectedID ? classes.selectedRoom : ''}`}>
                      <ListItemIcon><div className={classes.roomAvatar}><RoomIcon /></div></ListItemIcon>
                      <ListItemText primary={room.name} secondary={`${room.onlineCount} 人在线 · ${room.playbackMode === 'fifo' ? '顺序队列' : '公平随机'}`} />
                      <Chip size="small" label={room.status === 'open' ? '开放' : '已关闭'} color={room.status === 'open' ? 'primary' : 'default'} variant={room.status === 'open' ? 'outlined' : 'default'} />
                    </ListItem>
                  ))}
                </List>
              ) : (
                <div className={classes.empty}><div><HeadsetIcon style={{ fontSize: 54, opacity: .5 }} /><Typography variant="h6">还没有听歌房</Typography><Typography variant="body2">由 Navidrome 管理员创建第一个房间。</Typography></div></div>
              )}
            </Paper>
          </Grid>

          <Grid item xs={12} md={8}>
            <Paper className={classes.panel}>
              {!selectedRoom ? (
                <div className={classes.empty}><div><RoomIcon style={{ fontSize: 54, opacity: .5 }} /><Typography variant="h6">选择一个房间</Typography><Typography variant="body2">查看成员、邀请并生成分享二维码。</Typography></div></div>
              ) : (
                <>
                  <div className={classes.panelHeader}>
                    <div><Box display="flex" alignItems="center" style={{ gap: 8 }}><Typography variant="h5">{selectedRoom.name}</Typography><Chip size="small" color={selectedRoom.status === 'open' ? 'primary' : 'default'} label={selectedRoom.status === 'open' ? '开放' : '已关闭'} /></Box><Typography variant="caption" color="textSecondary">房间 ID：{selectedRoom.roomID}</Typography></div>
                    <div className={classes.grow} />
                    {isAdmin && <Tooltip title="房间设置"><IconButton onClick={() => setRoomDialog('edit')}><EditIcon /></IconButton></Tooltip>}
                  </div>
                  <Divider />
                  <div className={classes.detailBody}>
                    <Grid container spacing={2}>
                      <Grid item xs={6} sm={3}><div className={classes.metric}><Typography variant="caption" color="textSecondary">在线</Typography><Typography variant="h6">{selectedRoom.onlineCount}</Typography></div></Grid>
                      <Grid item xs={6} sm={3}><div className={classes.metric}><Typography variant="caption" color="textSecondary">成员</Typography><Typography variant="h6">{members.filter((member) => member.active).length}</Typography></div></Grid>
                      <Grid item xs={6} sm={3}><div className={classes.metric}><Typography variant="caption" color="textSecondary">有效邀请</Typography><Typography variant="h6">{activeInvites.length}</Typography></div></Grid>
                      <Grid item xs={6} sm={3}><div className={classes.metric}><Typography variant="caption" color="textSecondary">队列模式</Typography><Typography variant="h6">{selectedRoom.playbackMode === 'fifo' ? '顺序' : '随机'}</Typography></div></Grid>
                    </Grid>

                    {isAdmin && <div className={classes.detailActions}>
                      <Button variant="outlined" startIcon={<WebIcon />} disabled={selectedRoom.status !== 'open'} href={`../join/${selectedRoom.roomID}/`}>进入 Web 听歌房</Button>
                      <Button variant="contained" color="primary" startIcon={<ShareIcon />} disabled={selectedRoom.status !== 'open'} onClick={() => setShareOpen(true)}>分享链接 / 二维码</Button>
                      <Button variant="outlined" startIcon={<PowerIcon />} disabled={busy} onClick={toggleRoom}>{selectedRoom.status === 'open' ? '关闭房间' : '重新开启'}</Button>
                      <Button variant="outlined" className={classes.danger} startIcon={<DeleteIcon />} disabled={busy} onClick={deleteRoom}>删除</Button>
                    </div>}

                    <section className={classes.section}>
                      <div className={classes.sectionHeader}><GroupIcon color="primary" /><Typography variant="h6">成员</Typography>{detailsLoading && <CircularProgress size={16} />}</div>
                      <Divider />
                      <List dense>
                        {members.map((member) => (
                          <ListItem key={member.username}>
                            <ListItemIcon><div className={member.online ? classes.onlineDot : classes.offlineDot} /></ListItemIcon>
                            <ListItemText primary={member.displayName || member.username} secondary={`@${member.username} · ${member.role === 'owner' ? '房主' : '成员'} · ${member.online ? '在线' : `上次活动 ${formatDate(member.lastSeenAt)}`}`} />
                            {isAdmin && member.role !== 'owner' && <ListItemSecondaryAction><Tooltip title="移除成员"><IconButton edge="end" onClick={() => removeMember(member)} disabled={busy}><DeleteIcon /></IconButton></Tooltip></ListItemSecondaryAction>}
                          </ListItem>
                        ))}
                      </List>
                    </section>

                    <section className={classes.section}>
                      <div className={classes.sectionHeader}><LinkIcon color="primary" /><Typography variant="h6">邀请记录</Typography></div>
                      <Divider />
                      <List dense>
                        {invites.length ? invites.map((invite) => {
                          const active = !invite.revokedAt && new Date(invite.expiresAt) > new Date() && invite.useCount < invite.maxUses
                          return <ListItem key={invite.inviteID}><ListItemText primary={invite.label || '未命名邀请'} secondary={`${invite.useCount}/${invite.maxUses} 次 · 到期 ${formatDate(invite.expiresAt)}${invite.revokedAt ? ' · 已撤销' : ''}`} /><ListItemSecondaryAction>{active && isAdmin ? <Button size="small" onClick={() => revokeInvite(invite.inviteID)} disabled={busy}>撤销</Button> : <Chip size="small" label={active ? '有效' : '失效'} variant="outlined" />}</ListItemSecondaryAction></ListItem>
                        }) : <ListItem><ListItemText primary="暂无邀请" secondary="点击“分享链接 / 二维码”生成一条邀请。密钥不会被持久保存。" /></ListItem>}
                      </List>
                    </section>

                    <section className={`${classes.section} ${classes.roadmap}`}>
                      <div className={classes.sectionHeader}><CodeIcon /><Typography variant="subtitle1">开源路线图</Typography></div>
                      <Typography variant="body2" color="textSecondary">群聊和更多房间能力正在开发，将继续以 GPL-3.0 开源方式提供。欢迎在 GitHub 提交建议或参与贡献。</Typography>
                    </section>
                  </div>
                </>
              )}
            </Paper>
          </Grid>
        </Grid>
      </main>

      <RoomDialog open={Boolean(roomDialog)} room={roomDialog === 'edit' ? selectedRoom : null} folders={folders} busy={busy} onClose={() => setRoomDialog(null)} onSave={saveRoom} />
      <ShareDialog open={shareOpen} room={selectedRoom} busy={busy} onClose={() => setShareOpen(false)} onCreate={createInvite} onCopied={() => setNotice('已复制到剪贴板')} />
      <Snackbar className={classes.snackbar} open={Boolean(notice)} autoHideDuration={3500} onClose={() => setNotice('')} message={notice} action={<IconButton size="small" color="inherit" onClick={() => setNotice('')}><CloseIcon fontSize="small" /></IconButton>} />
      <Snackbar className={classes.snackbar} open={Boolean(error)} autoHideDuration={7000} onClose={() => setError('')} message={error} action={<IconButton size="small" color="inherit" onClick={() => setError('')}><CloseIcon fontSize="small" /></IconButton>} />
    </div>
  )
}

export default App
