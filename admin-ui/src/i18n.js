export const locales = {
  en: 'English',
  'zh-CN': '简体中文',
  'zh-TW': '繁體中文',
  ja: '日本語',
  es: 'Español',
  fr: 'Français',
  de: 'Deutsch',
}

// Columns follow the locale order above; keeping each message together makes omissions visible.
const strings = {
  appName: ['Music Room', '听歌房', '聽歌房', '音楽ルーム', 'Sala de música', 'Salon musical', 'Musikraum'],
  pageTitle: ['Music Room · Navidrome', '听歌房 · Navidrome', '聽歌房 · Navidrome', '音楽ルーム · Navidrome', 'Sala de música · Navidrome', 'Salon musical · Navidrome', 'Musikraum · Navidrome'],
  language: ['Language', '语言', '語言', '言語', 'Idioma', 'Langue', 'Sprache'],
  pluginLabel: ['Navidrome Plugin', 'Navidrome 插件', 'Navidrome 外掛', 'Navidrome プラグイン', 'Complemento de Navidrome', 'Extension Navidrome', 'Navidrome-Plugin'],
  back: ['Return to Navidrome', '返回 Navidrome', '返回 Navidrome', 'Navidrome に戻る', 'Volver a Navidrome', 'Retour à Navidrome', 'Zurück zu Navidrome'],
  reload: ['Reload', '重新加载', '重新載入', '再読み込み', 'Recargar', 'Recharger', 'Neu laden'],
  refresh: ['Refresh', '刷新', '重新整理', '更新', 'Actualizar', 'Actualiser', 'Aktualisieren'],
  loading: ['Loading…', '加载中…', '載入中…', '読み込み中…', 'Cargando…', 'Chargement…', 'Wird geladen…'],
  recoveryTitle: ['The page could not be displayed', '页面无法显示', '頁面無法顯示', 'ページを表示できません', 'No se pudo mostrar la página', 'Impossible d’afficher la page', 'Die Seite konnte nicht angezeigt werden'],
  recoveryBody: ['Reload to try again, or return to Navidrome.', '请重新加载，或返回 Navidrome。', '請重新載入，或返回 Navidrome。', '再読み込みするか、Navidrome に戻ってください。', 'Recarga la página o vuelve a Navidrome.', 'Rechargez la page ou retournez à Navidrome.', 'Lade die Seite neu oder kehre zu Navidrome zurück.'],
  loginTitle: ['Sign in to Navidrome first', '请先登录 Navidrome', '請先登入 Navidrome', '先に Navidrome にログインしてください', 'Primero inicia sesión en Navidrome', 'Connectez-vous d’abord à Navidrome', 'Melde dich zuerst bei Navidrome an'],
  loginHelp: ['This page reuses your Navidrome OpenSubsonic credentials. After signing in, open Settings → Plugins → Navidrome Music Room → Website.', '管理页会安全复用当前 Navidrome 登录的 OpenSubsonic 凭据。请登录后，从“设置 → 插件 → Navidrome Music Room → Website”重新打开。', '管理頁會安全重用目前 Navidrome 登入的 OpenSubsonic 憑證。登入後，請從「設定 → 外掛 → Navidrome Music Room → Website」重新開啟。', 'このページは Navidrome の OpenSubsonic 認証情報を使用します。ログイン後、設定 → プラグイン → Navidrome Music Room → Website から開いてください。', 'Esta página utiliza tus credenciales OpenSubsonic de Navidrome. Tras iniciar sesión, abre Ajustes → Complementos → Navidrome Music Room → Website.', 'Cette page utilise vos identifiants OpenSubsonic de Navidrome. Après connexion, ouvrez Paramètres → Extensions → Navidrome Music Room → Website.', 'Diese Seite verwendet deine OpenSubsonic-Zugangsdaten von Navidrome. Öffne nach der Anmeldung Einstellungen → Plugins → Navidrome Music Room → Website.'],
  sessionFailed: ['Could not start a room session: {error}', '无法建立听歌房会话：{error}', '無法建立聽歌房工作階段：{error}', 'ルームセッションを開始できません：{error}', 'No se pudo iniciar la sesión: {error}', 'Impossible de démarrer la session : {error}', 'Raumsitzung konnte nicht gestartet werden: {error}'],
  connected: ['Room service connected', '房间服务已连接', '房間服務已連線', 'ルームサービスに接続済み', 'Servicio de salas conectado', 'Service de salons connecté', 'Raumdienst verbunden'],
  gatewayAddress: ['Gateway {url}', '网关 {url}', '閘道 {url}', 'ゲートウェイ {url}', 'Gateway {url}', 'Passerelle {url}', 'Gateway {url}'],
  openRooms: ['Open rooms: {count}', '{count} 个房间开启中', '{count} 個房間開放中', '公開中のルーム：{count}', 'Salas abiertas: {count}', 'Salons ouverts : {count}', 'Offene Räume: {count}'],
  mediaHelp: ['Each member streams directly from Navidrome using their own account.', '媒体仍由各成员账号直接从 Navidrome 播放', '媒體仍由各成員帳號直接從 Navidrome 播放', '各メンバーのアカウントで Navidrome から直接再生します。', 'Cada miembro reproduce directamente desde Navidrome con su propia cuenta.', 'Chaque membre écoute directement depuis Navidrome avec son propre compte.', 'Jedes Mitglied streamt mit dem eigenen Konto direkt von Navidrome.'],
  notAdmin: ['You can view rooms you have joined. A Navidrome administrator is required to create or manage rooms.', '当前账号不是 Navidrome 管理员，可以查看已加入的房间，但不能创建或管理房间。', '目前帳號不是 Navidrome 管理員，可以查看已加入的房間，但不能建立或管理房間。', '参加済みのルームを閲覧できます。作成や管理には Navidrome 管理者権限が必要です。', 'Puedes ver las salas a las que te has unido. Solo un administrador de Navidrome puede crearlas o gestionarlas.', 'Vous pouvez consulter vos salons. Leur création et leur gestion nécessitent un administrateur Navidrome.', 'Du kannst beigetretene Räume ansehen. Zum Erstellen und Verwalten sind Navidrome-Administratorrechte erforderlich.'],
  rooms: ['Rooms', '房间', '房間', 'ルーム', 'Salas', 'Salons', 'Räume'],
  visibleRooms: ['Visible rooms: {count}', '{count} 个可见房间', '{count} 個可見房間', '表示可能なルーム：{count}', 'Salas visibles: {count}', 'Salons visibles : {count}', 'Sichtbare Räume: {count}'],
  create: ['Create', '创建', '建立', '作成', 'Crear', 'Créer', 'Erstellen'],
  onlineCount: ['{count} online', '{count} 人在线', '{count} 人在線', '{count} 人オンライン', '{count} en línea', '{count} en ligne', '{count} online'],
  open: ['Open', '开放', '開放', '公開中', 'Abierta', 'Ouvert', 'Offen'],
  closed: ['Closed', '已关闭', '已關閉', '閉鎖中', 'Cerrada', 'Fermé', 'Geschlossen'],
  fifo: ['In request order', '按点歌顺序', '依點歌順序', 'リクエスト順', 'Por orden de solicitud', 'Par ordre de demande', 'In Wunschreihenfolge'],
  fair_random: ['Fair shuffle', '公平随机', '公平隨機', '公平なシャッフル', 'Aleatorio equitativo', 'Aléatoire équitable', 'Faire Zufallswiedergabe'],
  noRooms: ['No rooms yet', '还没有听歌房', '尚無聽歌房', 'ルームがありません', 'Aún no hay salas', 'Aucun salon pour le moment', 'Noch keine Räume'],
  noRoomsHelp: ['A Navidrome administrator can create the first room.', '由 Navidrome 管理员创建第一个房间。', '由 Navidrome 管理員建立第一個房間。', 'Navidrome 管理者が最初のルームを作成できます。', 'Un administrador de Navidrome puede crear la primera sala.', 'Un administrateur Navidrome peut créer le premier salon.', 'Ein Navidrome-Administrator kann den ersten Raum erstellen.'],
  selectRoom: ['Select a room', '选择一个房间', '選擇房間', 'ルームを選択', 'Selecciona una sala', 'Sélectionnez un salon', 'Wähle einen Raum'],
  selectRoomHelp: ['View members and invitations, and share a QR code.', '查看成员、邀请并生成分享二维码。', '查看成員、邀請並產生分享 QR 碼。', 'メンバーや招待の確認、共有 QR コードの作成ができます。', 'Consulta miembros e invitaciones y comparte un código QR.', 'Consultez les membres et invitations, et partagez un code QR.', 'Mitglieder und Einladungen ansehen und einen QR-Code teilen.'],
  roomID: ['Room ID: {id}', '房间 ID：{id}', '房間 ID：{id}', 'ルーム ID：{id}', 'ID de sala: {id}', 'ID du salon : {id}', 'Raum-ID: {id}'],
  roomSettings: ['Room settings', '房间设置', '房間設定', 'ルーム設定', 'Ajustes de la sala', 'Paramètres du salon', 'Raumeinstellungen'],
  createRoom: ['Create room', '创建房间', '建立房間', 'ルームを作成', 'Crear sala', 'Créer un salon', 'Raum erstellen'],
  roomName: ['Room name', '房间名称', '房間名稱', 'ルーム名', 'Nombre de la sala', 'Nom du salon', 'Raumname'],
  queueLimit: ['Pending tracks per member', '每人待播上限', '每人待播上限', 'メンバーごとの待機曲数上限', 'Canciones pendientes por miembro', 'Titres en attente par membre', 'Wartende Titel pro Mitglied'],
  queueMode: ['Queue mode', '队列模式', '佇列模式', 'キューモード', 'Modo de cola', 'Mode de file', 'Wiedergabemodus'],
  libraries: ['Available music libraries', '可使用的音乐库', '可使用的音樂庫', '利用可能な音楽ライブラリ', 'Bibliotecas de música disponibles', 'Bibliothèques musicales disponibles', 'Verfügbare Musikbibliotheken'],
  musicFolder: ['Music library {id}', '音乐库 {id}', '音樂庫 {id}', '音楽ライブラリ {id}', 'Biblioteca de música {id}', 'Bibliothèque musicale {id}', 'Musikbibliothek {id}'],
  noLibraries: ['Your account has no available Navidrome music libraries.', '当前账号没有可用的 Navidrome 音乐库。', '目前帳號沒有可用的 Navidrome 音樂庫。', 'このアカウントで利用できる Navidrome 音楽ライブラリはありません。', 'Tu cuenta no tiene bibliotecas de música disponibles en Navidrome.', 'Votre compte n’a aucune bibliothèque musicale Navidrome disponible.', 'Für dein Konto sind keine Navidrome-Musikbibliotheken verfügbar.'],
  preload: ['Preload next track', '预加载下一首', '預載下一首', '次の曲を先読み', 'Precargar la siguiente canción', 'Précharger le titre suivant', 'Nächsten Titel vorladen'],
  allowMemberSkip: ['Allow room members to skip tracks', '允许房间成员切歌', '允許房間成員切歌', 'ルームメンバーによる曲のスキップを許可', 'Permitir que los miembros salten canciones', 'Autoriser les membres à passer les titres', 'Mitglieder dürfen Titel überspringen'],
  skipPolicy: ['Track skipping', '切歌权限', '切歌權限', '曲のスキップ', 'Saltar canciones', 'Passage des titres', 'Titel überspringen'],
  allMembers: ['All members', '所有成员', '所有成員', '全メンバー', 'Todos los miembros', 'Tous les membres', 'Alle Mitglieder'],
  managersOnly: ['Managers only', '仅管理员', '僅管理員', '管理者のみ', 'Solo gestores', 'Gestionnaires uniquement', 'Nur Raumverwalter'],
  cancel: ['Cancel', '取消', '取消', 'キャンセル', 'Cancelar', 'Annuler', 'Abbrechen'],
  save: ['Save', '保存', '儲存', '保存', 'Guardar', 'Enregistrer', 'Speichern'],
  done: ['Done', '完成', '完成', '完了', 'Listo', 'Terminé', 'Fertig'],
  close: ['Close', '关闭', '關閉', '閉じる', 'Cerrar', 'Fermer', 'Schließen'],
  online: ['Online', '在线', '在線', 'オンライン', 'En línea', 'En ligne', 'Online'],
  members: ['Members', '成员', '成員', 'メンバー', 'Miembros', 'Membres', 'Mitglieder'],
  member: ['Member', '成员', '成員', 'メンバー', 'Miembro', 'Membre', 'Mitglied'],
  owner: ['Owner', '房主', '房主', 'オーナー', 'Propietario', 'Propriétaire', 'Besitzer'],
  activeInvites: ['Active invitations', '有效邀请', '有效邀請', '有効な招待', 'Invitaciones activas', 'Invitations actives', 'Aktive Einladungen'],
  enterRoom: ['Open web music room', '进入 Web 听歌房', '進入 Web 聽歌房', 'Web 音楽ルームを開く', 'Abrir sala web', 'Ouvrir le salon web', 'Web-Musikraum öffnen'],
  share: ['Share link / QR code', '分享链接 / 二维码', '分享連結 / QR 碼', 'リンク・QR コードを共有', 'Compartir enlace / QR', 'Partager un lien / QR code', 'Link / QR-Code teilen'],
  closeRoom: ['Close room', '关闭房间', '關閉房間', 'ルームを閉鎖', 'Cerrar sala', 'Fermer le salon', 'Raum schließen'],
  reopen: ['Reopen', '重新开启', '重新開放', '再公開', 'Reabrir', 'Rouvrir', 'Wieder öffnen'],
  delete: ['Delete', '删除', '刪除', '削除', 'Eliminar', 'Supprimer', 'Löschen'],
  deleteConfirm: ['Permanently delete “{name}”? Members, queue and history will also be deleted.', '确定永久删除「{name}」吗？房间成员、队列和历史也会被删除。', '確定永久刪除「{name}」嗎？房間成員、佇列和歷史也會被刪除。', '「{name}」を完全に削除しますか？メンバー、キュー、履歴も削除されます。', '¿Eliminar “{name}” permanentemente? También se eliminarán los miembros, la cola y el historial.', 'Supprimer définitivement « {name} » ? Les membres, la file et l’historique seront aussi supprimés.', '„{name}“ endgültig löschen? Mitglieder, Warteschlange und Verlauf werden ebenfalls gelöscht.'],
  roomSaved: ['Room settings saved', '房间设置已保存', '房間設定已儲存', 'ルーム設定を保存しました', 'Ajustes guardados', 'Paramètres enregistrés', 'Raumeinstellungen gespeichert'],
  roomCreated: ['Room created', '听歌房已创建', '聽歌房已建立', 'ルームを作成しました', 'Sala creada', 'Salon créé', 'Raum erstellt'],
  roomClosed: ['Room closed', '房间已关闭', '房間已關閉', 'ルームを閉鎖しました', 'Sala cerrada', 'Salon fermé', 'Raum geschlossen'],
  roomReopened: ['Room reopened', '房间已重新开启', '房間已重新開放', 'ルームを再公開しました', 'Sala reabierta', 'Salon rouvert', 'Raum wieder geöffnet'],
  roomDeleted: ['Room deleted', '房间已删除', '房間已刪除', 'ルームを削除しました', 'Sala eliminada', 'Salon supprimé', 'Raum gelöscht'],
  lastSeen: ['Last active {date}', '上次活动 {date}', '上次活動 {date}', '最終活動 {date}', 'Última actividad {date}', 'Dernière activité {date}', 'Zuletzt aktiv {date}'],
  removeMember: ['Remove member', '移除成员', '移除成員', 'メンバーを削除', 'Eliminar miembro', 'Retirer le membre', 'Mitglied entfernen'],
  removeConfirm: ['Remove {name} from this room?', '从房间移除 {name}？', '從房間移除 {name}？', '{name} をこのルームから削除しますか？', '¿Eliminar a {name} de esta sala?', 'Retirer {name} de ce salon ?', '{name} aus diesem Raum entfernen?'],
  memberRemoved: ['Member removed', '成员已移除', '成員已移除', 'メンバーを削除しました', 'Miembro eliminado', 'Membre retiré', 'Mitglied entfernt'],
  inviteHistory: ['Invitation history', '邀请记录', '邀請紀錄', '招待履歴', 'Historial de invitaciones', 'Historique des invitations', 'Einladungsverlauf'],
  unnamedInvite: ['Unnamed invitation', '未命名邀请', '未命名邀請', '名前のない招待', 'Invitación sin nombre', 'Invitation sans nom', 'Unbenannte Einladung'],
  inviteUsage: ['{count}/{max} uses · Expires {date}', '{count}/{max} 次 · 到期 {date}', '{count}/{max} 次 · 到期 {date}', '{count}/{max} 回使用 · 有効期限 {date}', '{count}/{max} usos · Caduca {date}', '{count}/{max} utilisations · Expire le {date}', '{count}/{max} Nutzungen · Läuft ab {date}'],
  revoked: ['Revoked', '已撤销', '已撤銷', '無効化済み', 'Revocada', 'Révoquée', 'Widerrufen'],
  revoke: ['Revoke', '撤销', '撤銷', '無効化', 'Revocar', 'Révoquer', 'Widerrufen'],
  active: ['Active', '有效', '有效', '有効', 'Activa', 'Active', 'Gültig'],
  inactive: ['Inactive', '失效', '失效', '無効', 'Inactiva', 'Inactive', 'Ungültig'],
  noInvites: ['No invitations yet', '暂无邀请', '尚無邀請', '招待がありません', 'Aún no hay invitaciones', 'Aucune invitation', 'Noch keine Einladungen'],
  noInvitesHelp: ['Use “Share link / QR code” to create an invitation. The secret is not stored.', '点击“分享链接 / 二维码”生成一条邀请。密钥不会被持久保存。', '點選「分享連結 / QR 碼」產生邀請。密鑰不會被永久儲存。', '「リンク・QR コードを共有」で招待を作成します。秘密鍵は保存されません。', 'Usa “Compartir enlace / QR” para crear una invitación. La clave no se guarda.', 'Utilisez « Partager un lien / QR code » pour créer une invitation. La clé n’est pas conservée.', 'Erstelle über „Link / QR-Code teilen“ eine Einladung. Der Schlüssel wird nicht gespeichert.'],
  inviteRevoked: ['Invitation revoked; existing members are unaffected', '邀请已撤销；已经加入的成员不受影响', '邀請已撤銷；已加入的成員不受影響', '招待を無効化しました。参加済みメンバーには影響しません', 'Invitación revocada; los miembros actuales no se ven afectados', 'Invitation révoquée ; les membres actuels ne sont pas affectés', 'Einladung widerrufen; bestehende Mitglieder bleiben erhalten'],
  shareTitle: ['Share “{name}”', '分享「{name}」', '分享「{name}」', '「{name}」を共有', 'Compartir “{name}”', 'Partager « {name} »', '„{name}“ teilen'],
  shareInvite: ['Shared invitation', '分享邀请', '分享邀請', '共有招待', 'Invitación compartida', 'Invitation partagée', 'Geteilte Einladung'],
  inviteHelp: ['Create a revocable invitation. Its secret is shown only once; the QR code is generated locally in your browser.', '创建一条可撤销的邀请。密钥只显示这一次，二维码在当前浏览器本地生成。', '建立可撤銷的邀請。密鑰只顯示一次，QR 碼在目前瀏覽器本機產生。', '無効化できる招待を作成します。秘密鍵は一度だけ表示され、QR コードはブラウザー内で生成されます。', 'Crea una invitación revocable. La clave se muestra una sola vez y el QR se genera en tu navegador.', 'Créez une invitation révocable. La clé n’est affichée qu’une fois ; le QR code est généré dans votre navigateur.', 'Erstelle eine widerrufbare Einladung. Der Schlüssel wird nur einmal angezeigt; der QR-Code entsteht lokal im Browser.'],
  inviteLabel: ['Invitation note', '邀请备注', '邀請備註', '招待のメモ', 'Nota de invitación', 'Note de l’invitation', 'Einladungsnotiz'],
  inviteDays: ['Valid for (days)', '有效天数', '有效天數', '有効日数', 'Validez (días)', 'Validité (jours)', 'Gültigkeit (Tage)'],
  maxUses: ['Maximum uses', '最多使用次数', '最多使用次數', '使用回数の上限', 'Usos máximos', 'Utilisations maximales', 'Maximale Nutzungen'],
  singleUse: ['Allow one use only', '仅允许使用一次', '僅允許使用一次', '一度だけ使用可能', 'Permitir un solo uso', 'Autoriser une seule utilisation', 'Nur einmal nutzbar'],
  inviteCreated: ['Invitation created', '邀请已生成', '邀請已產生', '招待を作成しました', 'Invitación creada', 'Invitation créée', 'Einladung erstellt'],
  qrHelp: ['Scan or open the link to join the web room or MusicMate. The invitation secret stays in the URL fragment, outside server access logs.', '扫码或打开链接可直接进入 Web 听歌房，也可以交给 MusicMate 处理。邀请密钥位于 URL fragment，不会进入服务器访问日志。', '掃碼或開啟連結可進入 Web 聽歌房，也可由 MusicMate 處理。邀請密鑰位於 URL fragment，不會進入伺服器存取紀錄。', 'スキャンまたはリンクで Web ルームや MusicMate に参加できます。招待の秘密鍵は URL フラグメントに含まれ、サーバーログには記録されません。', 'Escanea o abre el enlace para entrar en la sala web o MusicMate. La clave queda en el fragmento de la URL, fuera de los registros del servidor.', 'Scannez ou ouvrez le lien pour rejoindre le salon web ou MusicMate. La clé reste dans le fragment de l’URL, hors des journaux du serveur.', 'Scanne den Code oder öffne den Link für den Web-Raum oder MusicMate. Der Schlüssel bleibt im URL-Fragment und erscheint nicht in Server-Zugriffsprotokollen.'],
  qrTitle: ['Room invitation QR code', '房间邀请二维码', '房間邀請 QR 碼', 'ルーム招待 QR コード', 'Código QR de invitación', 'QR code d’invitation', 'QR-Code zur Raumeinladung'],
  copyLink: ['Copy room link', '复制房间链接', '複製房間連結', 'ルームリンクをコピー', 'Copiar enlace', 'Copier le lien du salon', 'Raumlink kopieren'],
  copyDeepLink: ['Copy app link', '复制 Deep Link', '複製 Deep Link', 'アプリリンクをコピー', 'Copiar enlace de la app', 'Copier le lien de l’application', 'App-Link kopieren'],
  generateInvite: ['Generate link and QR code', '生成链接和二维码', '產生連結和 QR 碼', 'リンクと QR コードを生成', 'Generar enlace y QR', 'Générer le lien et le QR code', 'Link und QR-Code erstellen'],
  copied: ['Copied to clipboard', '已复制到剪贴板', '已複製到剪貼簿', 'クリップボードにコピーしました', 'Copiado al portapapeles', 'Copié dans le presse-papiers', 'In die Zwischenablage kopiert'],
  roadmap: ['Open-source roadmap', '开源路线图', '開源路線圖', 'オープンソースの開発計画', 'Plan de código abierto', 'Feuille de route open source', 'Open-Source-Roadmap'],
  roadmapHelp: ['Synchronized playback, chat, statistics, rankings, achievements, and wishes are available under GPL-3.0. Native Navidrome context actions and multi-node support remain on the open-source roadmap.', '同步播放、群聊、统计、排行榜、成就和心愿已按 GPL-3.0 开源提供。Navidrome 原生上下文操作和多节点支持仍在开源路线图中。', '同步播放、群聊、統計、排行榜、成就和心願已按 GPL-3.0 開源提供。Navidrome 原生內容選單操作和多節點支援仍在開源路線圖中。', '同期再生、チャット、統計、ランキング、実績、リクエストは GPL-3.0 で公開済みです。Navidrome 標準のコンテキスト操作とマルチノード対応は、引き続きオープンソースのロードマップに含まれています。', 'La reproducción sincronizada, el chat, las estadísticas, las clasificaciones, los logros y los deseos ya están disponibles bajo GPL-3.0. Las acciones contextuales nativas de Navidrome y el soporte multinodo siguen en la hoja de ruta de código abierto.', 'La lecture synchronisée, le chat, les statistiques, les classements, les succès et les souhaits sont disponibles sous GPL-3.0. Les actions contextuelles natives de Navidrome et la prise en charge de plusieurs nœuds restent sur la feuille de route open source.', 'Synchronisierte Wiedergabe, Chat, Statistiken, Ranglisten, Erfolge und Wünsche sind unter GPL-3.0 verfügbar. Native Navidrome-Kontextaktionen und die Unterstützung mehrerer Knoten stehen weiterhin auf der Open-Source-Roadmap.'],
  updates: ['Updates & System Health', '更新与系统健康', '更新與系統健康', '更新とシステム状態', 'Actualizaciones y estado del sistema', 'Mises à jour et état du système', 'Updates und Systemzustand'],
  gatewayVersion: ['Gateway version', '网关版本', '閘道版本', 'ゲートウェイのバージョン', 'Versión del gateway', 'Version de la passerelle', 'Gateway-Version'],
  pluginVersion: ['Paired .ndp plugin version', '配对的 .ndp 插件版本', '配對的 .ndp 外掛版本', 'ペアリング済み .ndp バージョン', 'Versión del complemento .ndp vinculado', 'Version de l’extension .ndp associée', 'Version des gekoppelten .ndp-Plugins'],
  pairing: ['Pairing status', '配对状态', '配對狀態', 'ペアリング状態', 'Estado de vinculación', 'État de l’association', 'Kopplungsstatus'],
  paired: ['Paired', '已配对', '已配對', 'ペアリング済み', 'Vinculado', 'Associé', 'Gekoppelt'],
  stale: ['Heartbeat stale', '心跳已过期', '心跳已過期', 'ハートビート期限切れ', 'Señal de actividad caducada', 'Signal d’activité périmé', 'Heartbeat veraltet'],
  not_paired: ['Not paired', '尚未配对', '尚未配對', '未ペアリング', 'Sin vincular', 'Non associé', 'Nicht gekoppelt'],
  unknown: ['Unknown', '未知', '未知', '不明', 'Desconocido', 'Inconnu', 'Unbekannt'],
  versionMismatch: ['Gateway and plugin versions differ', '网关与插件版本不一致', '閘道與外掛版本不一致', 'ゲートウェイとプラグインのバージョンが異なります', 'Las versiones del gateway y del complemento no coinciden', 'Les versions de la passerelle et de l’extension diffèrent', 'Gateway- und Plugin-Version stimmen nicht überein'],
  channel: ['Update channel', '更新通道', '更新通道', '更新チャンネル', 'Canal de actualización', 'Canal de mise à jour', 'Update-Kanal'],
  stable: ['Stable', '稳定版', '穩定版', '安定版', 'Estable', 'Stable', 'Stabil'],
  beta: ['Beta', '测试版', '測試版', 'ベータ版', 'Beta', 'Bêta', 'Beta'],
  latestVersion: ['Latest detected version', '最新检测版本', '最新偵測版本', '検出された最新バージョン', 'Última versión detectada', 'Dernière version détectée', 'Zuletzt erkannte Version'],
  publishedAt: ['Release published', '发布于', '發佈於', 'リリース公開日', 'Fecha de publicación', 'Date de publication', 'Veröffentlicht am'],
  updateAvailable: ['Update available', '有可用更新', '有可用更新', '更新があります', 'Actualización disponible', 'Mise à jour disponible', 'Update verfügbar'],
  releaseNotes: ['Release notes', '更新说明', '更新說明', 'リリースノート', 'Notas de la versión', 'Notes de version', 'Versionshinweise'],
  noReleaseNotes: ['No release notes provided. See the GitHub release for details.', '未提供更新说明，请查看 GitHub 发布页面。', '未提供更新說明，請查看 GitHub 發佈頁面。', 'リリースノートはありません。GitHub のリリースをご確認ください。', 'No hay notas disponibles. Consulta la versión en GitHub.', 'Aucune note disponible. Consultez la version sur GitHub.', 'Keine Versionshinweise vorhanden. Details findest du auf GitHub.'],
  releasePage: ['View GitHub release', '查看 GitHub 发布页面', '查看 GitHub 發佈頁面', 'GitHub リリースを表示', 'Ver versión en GitHub', 'Voir la version sur GitHub', 'GitHub-Release ansehen'],
  checkUpdates: ['Check for updates', '检查更新', '檢查更新', '更新を確認', 'Buscar actualizaciones', 'Rechercher des mises à jour', 'Nach Updates suchen'],
  upgradeNow: ['Upgrade now', '立即升级', '立即升級', '今すぐ更新', 'Actualizar ahora', 'Mettre à jour maintenant', 'Jetzt aktualisieren'],
  rescanPlugin: ['Rescan & enable plugin in Navidrome', '在 Navidrome 重新扫描并启用插件', '在 Navidrome 重新掃描並啟用外掛', 'Navidrome でプラグインを再スキャンして有効化', 'Volver a escanear y activar el complemento en Navidrome', 'Analyser et activer l’extension dans Navidrome', 'Plugin in Navidrome neu einlesen und aktivieren'],
  rollback: ['Rollback', '回滚', '回復舊版', 'ロールバック', 'Revertir versión', 'Revenir à la version précédente', 'Zurückrollen'],
  diagnostics: ['Download diagnostics', '下载诊断信息', '下載診斷資訊', '診断情報をダウンロード', 'Descargar diagnóstico', 'Télécharger le diagnostic', 'Diagnose herunterladen'],
  manualUpgrade: ['This gateway is not managed by the launcher. For Docker Compose, run the command below. Otherwise install the release’s gateway binary and replace navidrome-music-room.ndp, then rescan and enable the plugin in Navidrome.', '此网关未由启动器管理。Docker Compose 部署请运行下方命令；其他部署请安装发布包中的网关程序并替换 navidrome-music-room.ndp，然后在 Navidrome 重新扫描并启用插件。', '此閘道未由啟動器管理。Docker Compose 部署請執行下方指令；其他部署請安裝發佈套件中的閘道程式並替換 navidrome-music-room.ndp，再於 Navidrome 重新掃描並啟用外掛。', 'このゲートウェイはランチャー管理外です。Docker Compose では以下のコマンドを実行してください。その他の場合はリリースのゲートウェイと navidrome-music-room.ndp を置き換え、Navidrome で再スキャンして有効化してください。', 'El launcher no gestiona este gateway. Con Docker Compose, ejecuta el comando siguiente. En otras instalaciones, instala el gateway de la versión y sustituye navidrome-music-room.ndp; después vuelve a escanear y activar el complemento.', 'Cette passerelle n’est pas gérée par le lanceur. Avec Docker Compose, exécutez la commande ci-dessous. Sinon, installez la passerelle publiée et remplacez navidrome-music-room.ndp, puis analysez et activez l’extension.', 'Dieses Gateway wird nicht vom Launcher verwaltet. Führe bei Docker Compose den folgenden Befehl aus. Installiere sonst das Gateway der Veröffentlichung und ersetze navidrome-music-room.ndp. Lies das Plugin danach in Navidrome neu ein und aktiviere es.'],
  upgradeConfirm: ['Upgrade to {version}? The gateway will restart and playback may be interrupted. Keep this page open to activate the plugin and confirm its heartbeat.', '升级到 {version}？网关将重启，播放可能中断。请保持此页打开，以启用插件并确认心跳。', '升級至 {version}？閘道將重新啟動，播放可能中斷。請保持此頁開啟，以啟用外掛並確認心跳。', '{version} に更新しますか？ゲートウェイが再起動し、再生が中断する場合があります。プラグインの有効化とハートビート確認のため、このページを開いたままにしてください。', '¿Actualizar a {version}? El gateway se reiniciará y la reproducción puede interrumpirse. Mantén esta página abierta para activar el complemento y confirmar su señal de actividad.', 'Mettre à jour vers {version} ? La passerelle redémarrera et la lecture pourra être interrompue. Gardez cette page ouverte pour activer l’extension et confirmer son signal d’activité.', 'Auf {version} aktualisieren? Das Gateway startet neu; die Wiedergabe kann unterbrochen werden. Lass diese Seite zur Plugin-Aktivierung und Heartbeat-Prüfung geöffnet.'],
  rollbackConfirm: ['Roll back to {version}? This restores the previous release AND its database backup. Room changes since that backup will be lost.', '回滚到 {version}？这将恢复旧版本及其数据库备份。备份之后的房间数据变更将丢失。', '回復至 {version}？這將還原舊版本及其資料庫備份。備份之後的房間資料變更將遺失。', '{version} に戻しますか？以前のリリースとデータベースのバックアップを復元します。バックアップ以降のルーム変更は失われます。', '¿Volver a {version}? Se restaurarán la versión anterior y su copia de la base de datos. Se perderán los cambios en las salas posteriores a esa copia.', 'Revenir à {version} ? La version précédente et sa sauvegarde de base de données seront restaurées. Les modifications des salons depuis cette sauvegarde seront perdues.', 'Auf {version} zurückrollen? Die vorherige Version UND ihre Datenbanksicherung werden wiederhergestellt. Raumänderungen seit dieser Sicherung gehen verloren.'],
  stepVerify: ['Verify Cosign/Sigstore signatures and stage release', '验证 Cosign/Sigstore 签名并暂存发布包', '驗證 Cosign/Sigstore 簽章並暫存發佈套件', 'Cosign/Sigstore 署名を検証してリリースを準備', 'Verificar firmas Cosign/Sigstore y preparar la versión', 'Vérifier les signatures Cosign/Sigstore et préparer la version', 'Cosign/Sigstore-Signaturen prüfen und Release bereitstellen'],
  stepRollback: ['Prepare the verified rollback and database backup', '准备已验证的回滚版本和数据库备份', '準備已驗證的舊版本和資料庫備份', '検証済みのロールバックとデータベースを準備', 'Preparar la reversión verificada y la copia de la base de datos', 'Préparer le retour vérifié et la sauvegarde de base de données', 'Geprüfte Rückkehr und Datenbanksicherung vorbereiten'],
  stepRestart: ['Restart gateway and swap release with launcher', '重启网关，由启动器切换版本', '重新啟動閘道，由啟動器切換版本', 'ゲートウェイを再起動し、ランチャーで切り替え', 'Reiniciar el gateway y cambiar la versión con el launcher', 'Redémarrer la passerelle et changer de version avec le lanceur', 'Gateway neu starten und Release durch Launcher wechseln'],
  stepReady: ['Wait for /readyz to report version {version}', '等待 /readyz 报告版本 {version}', '等待 /readyz 回報版本 {version}', '/readyz のバージョン {version} を待機', 'Esperar a que /readyz indique la versión {version}', 'Attendre la version {version} dans /readyz', 'Auf Version {version} von /readyz warten'],
  stepPlugin: ['Rescan, enable plugin and confirm matching paired heartbeat', '重新扫描并启用插件，确认版本匹配的配对心跳', '重新掃描並啟用外掛，確認版本相符的配對心跳', '再スキャンして有効化し、一致するペアリング済みハートビートを確認', 'Volver a escanear, activar y confirmar la vinculación y versión', 'Analyser, activer et confirmer l’association et la version', 'Plugin neu einlesen, aktivieren und passenden gekoppelten Heartbeat bestätigen'],
  upgradeComplete: ['Gateway and plugin are ready on {version}.', '网关与插件已就绪，版本为 {version}。', '閘道與外掛已就緒，版本為 {version}。', 'ゲートウェイとプラグインが {version} で準備完了しました。', 'El gateway y el complemento están listos en {version}.', 'La passerelle et l’extension sont prêtes en version {version}.', 'Gateway und Plugin sind mit Version {version} bereit.'],
  pluginActivated: ['Plugin enabled and paired with the matching version.', '插件已启用并完成配对，版本一致。', '外掛已啟用並完成配對，版本一致。', 'プラグインを有効化し、同じバージョンでペアリングしました。', 'Complemento activado y vinculado con la versión correcta.', 'Extension activée et associée avec la bonne version.', 'Plugin aktiviert und mit passender Version gekoppelt.'],
  updateFailed: ['The last update operation failed. Download diagnostics for details.', '上次更新操作失败，请下载诊断信息查看详情。', '上次更新操作失敗，請下載診斷資訊查看詳情。', '前回の更新に失敗しました。診断情報をダウンロードしてご確認ください。', 'La última actualización falló. Descarga el diagnóstico para más detalles.', 'La dernière mise à jour a échoué. Téléchargez le diagnostic pour plus de détails.', 'Der letzte Update-Vorgang ist fehlgeschlagen. Lade die Diagnose für Details herunter.'],
  'error.request_failed': ['The request failed. Please try again.', '请求失败，请重试。', '請求失敗，請重試。', 'リクエストに失敗しました。再試行してください。', 'La solicitud falló. Inténtalo de nuevo.', 'La requête a échoué. Réessayez.', 'Die Anfrage ist fehlgeschlagen. Bitte erneut versuchen.'],
  'error.room_closed': ['This room is closed.', '此房间已关闭。', '此房間已關閉。', 'このルームは閉鎖されています。', 'Esta sala está cerrada.', 'Ce salon est fermé.', 'Dieser Raum ist geschlossen.'],
  'error.membership_required': ['Join this room with an invitation first.', '请先通过邀请加入房间。', '請先透過邀請加入房間。', '先に招待でこのルームに参加してください。', 'Primero únete a la sala con una invitación.', 'Rejoignez d’abord ce salon avec une invitation.', 'Tritt diesem Raum zuerst über eine Einladung bei.'],
  'error.library_access_required': ['You do not have access to this music library.', '你没有此音乐库的访问权限。', '你沒有此音樂庫的存取權限。', 'この音楽ライブラリへのアクセス権がありません。', 'No tienes acceso a esta biblioteca de música.', 'Vous n’avez pas accès à cette bibliothèque musicale.', 'Du hast keinen Zugriff auf diese Musikbibliothek.'],
  'error.personal_queue_limit': ['You have reached your pending track limit.', '已达到每人待播上限。', '已達到每人待播上限。', '待機曲数の上限に達しました。', 'Has alcanzado tu límite de canciones pendientes.', 'Vous avez atteint votre limite de titres en attente.', 'Dein Limit für wartende Titel ist erreicht.'],
  'error.room_manager_required': ['Only room managers can do this.', '此操作需要房间管理权限。', '此操作需要房間管理權限。', 'ルーム管理者権限が必要です。', 'Esta acción requiere permisos de gestión de sala.', 'Cette action nécessite un gestionnaire du salon.', 'Dafür sind Raumverwalterrechte erforderlich.'],
  'error.skip_permission_required': ['You do not have permission to skip tracks.', '你没有切歌权限。', '你沒有切歌權限。', '曲をスキップする権限がありません。', 'No tienes permiso para saltar canciones.', 'Vous n’avez pas la permission de passer les titres.', 'Du darfst keine Titel überspringen.'],
  'error.launcher_required': ['Automatic upgrades require the Music Room launcher. Follow the manual upgrade instructions.', '自动升级需要听歌房启动器，请按手动升级说明操作。', '自動升級需要聽歌房啟動器，請依手動升級說明操作。', '自動更新には Music Room ランチャーが必要です。手動更新手順に従ってください。', 'La actualización automática requiere el launcher de Music Room. Sigue las instrucciones manuales.', 'Les mises à jour automatiques nécessitent le lanceur Music Room. Suivez les instructions manuelles.', 'Automatische Updates benötigen den Music-Room-Launcher. Folge der manuellen Anleitung.'],
  'error.plugin_not_paired': ['The Navidrome plugin is not paired. Rescan and enable it.', 'Navidrome 插件尚未配对，请重新扫描并启用。', 'Navidrome 外掛尚未配對，請重新掃描並啟用。', 'Navidrome プラグインが未ペアリングです。再スキャンして有効化してください。', 'El complemento de Navidrome no está vinculado. Vuelve a escanearlo y actívalo.', 'L’extension Navidrome n’est pas associée. Analysez-la et activez-la.', 'Das Navidrome-Plugin ist nicht gekoppelt. Lies es neu ein und aktiviere es.'],
  'error.plugin_lease_expired': ['The plugin heartbeat is stale. Rescan and enable the plugin.', '插件心跳已过期，请重新扫描并启用插件。', '外掛心跳已過期，請重新掃描並啟用外掛。', 'プラグインのハートビートが期限切れです。再スキャンして有効化してください。', 'La señal del complemento ha caducado. Vuelve a escanearlo y actívalo.', 'Le signal de l’extension est périmé. Analysez-la et activez-la.', 'Der Plugin-Heartbeat ist veraltet. Lies das Plugin neu ein und aktiviere es.'],
  'error.configuration_mismatch': ['The gateway and plugin configuration do not match. Check their shared settings.', '网关与插件配置不一致，请检查双方配置。', '閘道與外掛設定不一致，請檢查雙方設定。', 'ゲートウェイとプラグインの設定が一致しません。共通設定を確認してください。', 'La configuración del gateway y del complemento no coincide. Revisa los ajustes compartidos.', 'Les configurations de la passerelle et de l’extension diffèrent. Vérifiez leurs paramètres communs.', 'Gateway- und Plugin-Konfiguration stimmen nicht überein. Prüfe die gemeinsamen Einstellungen.'],
  'error.update_check_failed': ['Could not check for updates. Try again later.', '无法检查更新，请稍后重试。', '無法檢查更新，請稍後重試。', '更新を確認できません。後でもう一度お試しください。', 'No se pudieron buscar actualizaciones. Inténtalo más tarde.', 'Impossible de rechercher des mises à jour. Réessayez plus tard.', 'Updates konnten nicht geprüft werden. Versuche es später erneut.'],
  'error.rollback_unavailable': ['No verified rollback and matching database backup are available.', '没有可用的已验证回滚版本及匹配的数据库备份。', '沒有可用的已驗證舊版本及相符的資料庫備份。', '検証済みの旧バージョンと対応するデータベースバックアップがありません。', 'No hay una versión verificada y una copia de base de datos válidas para revertir.', 'Aucune version précédente vérifiée avec sauvegarde correspondante n’est disponible.', 'Keine geprüfte vorherige Version mit passender Datenbanksicherung verfügbar.'],
  'error.session_required': ['Your session is missing or expired. Reload or sign in to Navidrome again.', '会话缺失或已过期，请重新加载或重新登录 Navidrome。', '工作階段遺失或已過期，請重新載入或重新登入 Navidrome。', 'セッションがないか期限切れです。再読み込みするか Navidrome に再ログインしてください。', 'La sesión no existe o ha caducado. Recarga o inicia sesión de nuevo en Navidrome.', 'La session est absente ou expirée. Rechargez la page ou reconnectez-vous à Navidrome.', 'Deine Sitzung fehlt oder ist abgelaufen. Lade neu oder melde dich erneut bei Navidrome an.'],
  'error.navidrome_admin_required': ['A Navidrome administrator account is required.', '需要 Navidrome 管理员账号。', '需要 Navidrome 管理員帳號。', 'Navidrome 管理者アカウントが必要です。', 'Se requiere una cuenta de administrador de Navidrome.', 'Un compte administrateur Navidrome est requis.', 'Ein Navidrome-Administratorkonto ist erforderlich.'],
  'error.plugin_user_not_allowed': ['This account is not authorized to use the plugin.', '此账号未获授权使用插件。', '此帳號未獲授權使用外掛。', 'このアカウントはプラグインの利用を許可されていません。', 'Esta cuenta no está autorizada para usar el complemento.', 'Ce compte n’est pas autorisé à utiliser l’extension.', 'Dieses Konto darf das Plugin nicht verwenden.'],
  'error.navidrome_unavailable': ['Navidrome is unavailable or returned an invalid response.', 'Navidrome 不可用或响应无效。', 'Navidrome 無法使用或回應無效。', 'Navidrome に接続できないか、応答が無効です。', 'Navidrome no está disponible o devolvió una respuesta no válida.', 'Navidrome est indisponible ou a renvoyé une réponse invalide.', 'Navidrome ist nicht erreichbar oder hat ungültig geantwortet.'],
  'error.invalid_request': ['Some values are invalid. Check the form and try again.', '部分输入无效，请检查表单后重试。', '部分輸入無效，請檢查表單後重試。', '入力に誤りがあります。フォームを確認して再試行してください。', 'Algunos valores no son válidos. Revisa el formulario.', 'Certaines valeurs sont invalides. Vérifiez le formulaire.', 'Einige Werte sind ungültig. Prüfe das Formular.'],
  'error.not_found': ['The requested room, invitation or track no longer exists.', '请求的房间、邀请或歌曲已不存在。', '請求的房間、邀請或歌曲已不存在。', '指定したルーム、招待、曲は存在しません。', 'La sala, invitación o canción solicitada ya no existe.', 'Le salon, l’invitation ou le titre demandé n’existe plus.', 'Der angeforderte Raum, die Einladung oder der Titel existiert nicht mehr.'],
  'error.invite_invalid': ['This invitation is invalid, expired, revoked or fully used.', '此邀请无效、已过期、已撤销或已达到使用上限。', '此邀請無效、已過期、已撤銷或已達使用上限。', '招待が無効、期限切れ、無効化済み、または使用上限に達しています。', 'La invitación no es válida, ha caducado, fue revocada o agotó sus usos.', 'Cette invitation est invalide, expirée, révoquée ou épuisée.', 'Diese Einladung ist ungültig, abgelaufen, widerrufen oder aufgebraucht.'],
  'error.owner_cannot_be_removed': ['The room owner cannot be removed. Close or delete the room instead.', '无法移除房主，请关闭或删除房间。', '無法移除房主，請關閉或刪除房間。', 'オーナーは削除できません。ルームを閉鎖または削除してください。', 'No se puede eliminar al propietario. Cierra o elimina la sala.', 'Le propriétaire ne peut pas être retiré. Fermez ou supprimez le salon.', 'Der Besitzer kann nicht entfernt werden. Schließe oder lösche den Raum.'],
  'error.update_version_changed': ['The latest release changed. Check for updates again before upgrading.', '最新发布版本已变化，请重新检查更新后再升级。', '最新發佈版本已變更，請重新檢查更新後再升級。', '最新リリースが変わりました。更新を再確認してください。', 'La última versión ha cambiado. Busca actualizaciones de nuevo.', 'La dernière version a changé. Recherchez à nouveau les mises à jour.', 'Die neueste Version hat sich geändert. Suche vor dem Update erneut.'],
  'error.update_verification_failed': ['Release verification failed or required signed files are missing. The release was not installed.', '发布包验证失败或缺少必需的签名文件，未安装此版本。', '發佈套件驗證失敗或缺少必要的簽章檔案，未安裝此版本。', 'リリース検証に失敗したか、署名済みファイルがありません。インストールされていません。', 'La verificación falló o faltan archivos firmados. No se instaló la versión.', 'La vérification a échoué ou des fichiers signés manquent. La version n’a pas été installée.', 'Die Release-Prüfung ist fehlgeschlagen oder signierte Dateien fehlen. Die Version wurde nicht installiert.'],
  'error.update_download_failed': ['Could not download the release. Check connectivity and try again.', '无法下载发布包，请检查网络后重试。', '無法下載發佈套件，請檢查網路後重試。', 'リリースをダウンロードできません。接続を確認して再試行してください。', 'No se pudo descargar la versión. Revisa la conexión.', 'Impossible de télécharger la version. Vérifiez la connexion.', 'Das Release konnte nicht heruntergeladen werden. Prüfe die Verbindung.'],
  'error.missing_navidrome_token': ['Sign in to Navidrome in this browser, then rescan and enable the plugin.', '请先在此浏览器登录 Navidrome，然后重新扫描并启用插件。', '請先在此瀏覽器登入 Navidrome，再重新掃描並啟用外掛。', 'このブラウザーで Navidrome にログインしてから、プラグインを再スキャンして有効化してください。', 'Inicia sesión en Navidrome en este navegador y después vuelve a escanear y activar el complemento.', 'Connectez-vous à Navidrome dans ce navigateur, puis analysez et activez l’extension.', 'Melde dich in diesem Browser bei Navidrome an und lies das Plugin danach neu ein und aktiviere es.'],
  'error.navidrome_plugin_not_found': ['The Music Room .ndp plugin was not found in Navidrome. Install it and rescan.', 'Navidrome 中未找到听歌房 .ndp 插件，请安装后重新扫描。', 'Navidrome 中找不到聽歌房 .ndp 外掛，請安裝後重新掃描。', 'Navidrome に Music Room .ndp がありません。インストールして再スキャンしてください。', 'No se encontró el complemento .ndp de Music Room. Instálalo y vuelve a escanear.', 'L’extension .ndp Music Room est introuvable. Installez-la puis relancez l’analyse.', 'Das Music-Room-.ndp-Plugin wurde nicht gefunden. Installiere es und lies es neu ein.'],
  'error.navidrome_plugin_activation_failed': ['Navidrome could not rescan or enable the plugin. Check your administrator login and try again.', 'Navidrome 无法重新扫描或启用插件，请检查管理员登录后重试。', 'Navidrome 無法重新掃描或啟用外掛，請檢查管理員登入後重試。', 'プラグインの再スキャンまたは有効化に失敗しました。管理者ログインを確認してください。', 'No se pudo escanear o activar el complemento. Revisa tu sesión de administrador.', 'Impossible d’analyser ou d’activer l’extension. Vérifiez votre connexion administrateur.', 'Das Plugin konnte nicht neu eingelesen oder aktiviert werden. Prüfe deine Administratoranmeldung.'],
  'error.gateway_ready_timeout': ['The target gateway version did not become ready in time. Check system health before trying another update.', '目标网关版本未在规定时间内就绪，请检查系统状态后再尝试更新。', '目標閘道版本未在規定時間內就緒，請檢查系統狀態後再嘗試更新。', '指定バージョンの起動が時間切れになりました。再更新する前に状態を確認してください。', 'El gateway no estuvo listo a tiempo. Revisa el estado del sistema antes de otra actualización.', 'La version cible n’est pas devenue disponible à temps. Vérifiez l’état du système avant une nouvelle mise à jour.', 'Die Zielversion wurde nicht rechtzeitig bereit. Prüfe den Systemzustand vor einem weiteren Update.'],
  'error.plugin_ready_timeout': ['The matching plugin heartbeat was not confirmed. Rescan and enable the plugin, then check its configuration.', '未确认到版本匹配的插件心跳。请重新扫描并启用插件，然后检查配置。', '未確認到版本相符的外掛心跳。請重新掃描並啟用外掛，再檢查設定。', '一致するプラグインのハートビートを確認できません。再スキャンして有効化し、設定を確認してください。', 'No se confirmó la señal de la versión correcta del complemento. Vuelve a escanearlo, actívalo y revisa la configuración.', 'Le signal de l’extension correspondante n’a pas été confirmé. Analysez-la, activez-la et vérifiez sa configuration.', 'Der passende Plugin-Heartbeat wurde nicht bestätigt. Lies das Plugin neu ein, aktiviere es und prüfe die Konfiguration.'],
  'error.update_response_invalid': ['The server did not confirm the update target. Check system health before retrying.', '服务器未确认更新目标，请检查系统状态后重试。', '伺服器未確認更新目標，請檢查系統狀態後重試。', 'サーバーが更新先を確認しませんでした。状態を確認してから再試行してください。', 'El servidor no confirmó la versión de destino. Revisa el estado antes de reintentar.', 'Le serveur n’a pas confirmé la version cible. Vérifiez l’état avant de réessayer.', 'Der Server hat die Zielversion nicht bestätigt. Prüfe den Zustand vor einem erneuten Versuch.'],
  'error.database_unavailable': ['The room database is unavailable. Check gateway health.', '房间数据库不可用，请检查网关状态。', '房間資料庫無法使用，請檢查閘道狀態。', 'ルームのデータベースを利用できません。ゲートウェイの状態を確認してください。', 'La base de datos no está disponible. Revisa el gateway.', 'La base de données est indisponible. Vérifiez la passerelle.', 'Die Raumdatenbank ist nicht verfügbar. Prüfe das Gateway.'],
  'error.conflict': ['The data changed. Refresh and try again.', '数据已变化，请刷新后重试。', '資料已變更，請重新整理後重試。', 'データが変更されました。更新して再試行してください。', 'Los datos han cambiado. Actualiza e inténtalo de nuevo.', 'Les données ont changé. Actualisez et réessayez.', 'Die Daten haben sich geändert. Aktualisiere und versuche es erneut.'],
  'error.rate_limited': ['Too many requests. Wait a moment and try again.', '请求过于频繁，请稍后重试。', '請求過於頻繁，請稍後重試。', 'リクエストが多すぎます。しばらくして再試行してください。', 'Demasiadas solicitudes. Espera un momento.', 'Trop de requêtes. Patientez puis réessayez.', 'Zu viele Anfragen. Warte kurz und versuche es erneut.'],
  'error.queue_full': ['The room queue is full.', '房间队列已满。', '房間佇列已滿。', 'ルームのキューが満杯です。', 'La cola de la sala está llena.', 'La file du salon est pleine.', 'Die Warteschlange des Raums ist voll.'],
  'error.queue_empty': ['There is no track available to play.', '没有可播放的歌曲。', '沒有可播放的歌曲。', '再生できる曲がありません。', 'No hay canciones para reproducir.', 'Aucun titre à lire.', 'Es ist kein Titel zum Abspielen verfügbar.'],
  'error.copy_failed': ['Could not copy the link. Select and copy it manually.', '无法复制链接，请手动选择并复制。', '無法複製連結，請手動選取並複製。', 'リンクをコピーできません。手動で選択してコピーしてください。', 'No se pudo copiar. Selecciona y copia el enlace manualmente.', 'Impossible de copier. Sélectionnez et copiez le lien manuellement.', 'Der Link konnte nicht kopiert werden. Wähle und kopiere ihn manuell.'],
}

export const messages = Object.fromEntries(Object.keys(locales).map((locale, index) => [
  locale, Object.fromEntries(Object.entries(strings).map(([key, values]) => [key, values[index]])),
]))

function normalizeLocale(value) {
  if (typeof value !== 'string') return null
  const tag = value.trim().replace(/_/g, '-').toLowerCase()
  if (/^zh-(tw|hk|hant)(-|$)/.test(tag)) return 'zh-TW'
  if (tag.startsWith('zh')) return 'zh-CN'
  const language = tag.split('-')[0]
  return Object.hasOwn(locales, language) ? language : null
}

export function detectLocale(storage, navigatorObj = globalThis.navigator) {
  try {
    storage ??= globalThis.localStorage
    for (const key of ['nmr:locale', 'i18nextLng', 'locale']) {
      const locale = normalizeLocale(storage?.getItem(key))
      if (locale) return locale
    }
  } catch { /* Browser privacy settings may block storage. */ }
  const languages = Array.isArray(navigatorObj?.languages) ? navigatorObj.languages : []
  for (const language of [...languages, navigatorObj?.language]) {
    const locale = normalizeLocale(language)
    if (locale) return locale
  }
  return 'en'
}

export function translate(locale, key, params = {}) {
  const catalog = Object.hasOwn(messages, locale) ? messages[locale] : messages.en
  const message = (Object.hasOwn(catalog, key) && catalog[key]) || (Object.hasOwn(messages.en, key) && messages.en[key]) || key
  return message.replace(/\{(\w+)\}/g, (match, name) => params[name] ?? match)
}

const errorAliases = {
  session_expired: 'session_required', auth_proof_invalid: 'session_required', navidrome_auth_failed: 'session_required',
  navidrome_request_rejected: 'navidrome_unavailable', navidrome_bad_response: 'navidrome_unavailable',
  room_not_found: 'not_found', invite_not_found: 'not_found', track_not_found: 'not_found', queue_entry_not_found: 'not_found',
  invite_expired: 'invite_invalid', invite_revoked: 'invite_invalid', invite_exhausted: 'invite_invalid',
  owner_cannot_leave: 'owner_cannot_be_removed', queue_entry_owner_required: 'room_manager_required',
  room_queue_full: 'queue_full', no_current_track: 'queue_empty',
  revision_conflict: 'conflict', queue_order_stale: 'conflict', stale_plugin_generation: 'conflict', idempotency_key_reused: 'conflict',
  auth_rate_limited: 'rate_limited', idempotency_capacity_exceeded: 'rate_limited',
  plugin_pairing_failed: 'configuration_mismatch', origin_forbidden: 'configuration_mismatch',
  request_invalid: 'invalid_request', request_too_large: 'invalid_request', idempotency_key_invalid: 'invalid_request',
  room_id_invalid: 'invalid_request', room_name_invalid: 'invalid_request', queue_limit_invalid: 'invalid_request',
  playback_mode_invalid: 'invalid_request', queue_id_invalid: 'invalid_request', queue_order_invalid: 'invalid_request',
  invite_expiry_invalid: 'invalid_request', invite_id_invalid: 'invalid_request', invite_use_limit_invalid: 'invalid_request',
  username_invalid: 'invalid_request', track_invalid: 'invalid_request', plugin_sync_invalid: 'invalid_request',
  seek_invalid: 'invalid_request', playback_command_invalid: 'invalid_request', update_action_invalid: 'invalid_request',
  internal_error: 'request_failed', request_cancelled: 'request_failed', diagnostics_invalid: 'request_failed',
}
for (const code of ['asset_missing', 'asset_url_invalid', 'bundle_invalid', 'checksum_invalid', 'checksum_missing', 'signature_invalid', 'signature_missing', 'sbom_invalid', 'sbom_missing', 'provenance_invalid', 'provenance_missing', 'release_invalid']) {
  errorAliases[`update_${code}`] = 'update_verification_failed'
}

export function localizeError(locale, error) {
  const code = typeof error === 'string' ? error : error?.code || error?.reason
  const key = `error.${Object.hasOwn(errorAliases, code) ? errorAliases[code] : code}`
  return translate(locale, Object.hasOwn(messages.en, key) ? key : 'error.request_failed')
}

export function formatDate(value, locale) {
  const date = value ? new Date(value) : null
  if (!date || !Number.isFinite(date.getTime())) return '—'
  return new Intl.DateTimeFormat(locale, { dateStyle: 'medium', timeStyle: 'short' }).format(date)
}
