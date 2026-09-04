// 申请进度页词条（由 status 页改造时填充）
export default {
  // 页头与查询入口
  'status.title': '申请进度查询',
  'status.subtitle': '输入申请编号，查看审核、制卡、邮寄与签收状态',
  'status.requestCode': '申请编号',
  'status.requestCodePlaceholder': '例如: EXAB23CD9',
  'status.lookupBtn': '查询进度',
  'status.loading': '正在查询申请进度...',
  'status.failedTitle': '查询失败',
  'status.reenter': '重新输入',
  'status.notFound': '申请不存在或编号错误',

  // 详情基础字段
  'status.callSign': '呼号',
  'status.scene': '申请场景',
  'status.requestTime': '申请时间',
  'status.notice': '公告',

  // 审核结果
  'status.rejectedPrefix': '申请未通过：',
  'status.noReason': '未填写原因',

  // 物流查询
  'status.mailTracking': '物流查询',
  'status.kuaidiTip': '复制下方单号，到快递100 官网即可查询实时物流：',
  'status.openKuaidi': '在快递100 查询物流',
  'status.openKuaidiShort': '在快递100 查询',
  'status.copyTracking': '复制单号',
  'status.trackingCopied': '单号已复制',
  'status.copyTrackingFailed': '复制失败，请手动选择单号复制',

  // SWL 反寄
  'status.sendToAddress': '请将您的卡片寄至以下地址',
  'status.registerMailHead': '登记您的寄出信息',
  'status.registerMailTip': '您寄出收听卡后在此登记，我们会收到记录并尽快处理回寄。',
  'status.mailType': '邮寄方式',
  'status.registeredMail': '挂号信',
  'status.ordinaryMail': '平信',
  'status.trackingNo': '单号',
  'status.trackingPlaceholder': '您的挂号信/快递单号',
  'status.submitRegister': '提交登记',
  'status.trackingRequired': '挂号信请填写单号',
  'status.registerSuccess': '登记成功，我们已收到您的寄出记录',
  'status.registeredHead': '您的寄出登记',
  'status.trackingPrefix': '单号：',
  'status.registeredAtPrefix': '登记时间：',

  // 白名单引导
  'status.mailNotReceived': '邮件未收到？',
  'status.whitelistTipBefore': '审核通过、收卡、回寄卡片寄出等节点都会向您的邮箱发送提醒。若未收到，请检查垃圾邮件箱（Spam），并将发件邮箱',
  'status.whitelistTipAfter': '添加到白名单，以便后续通知正常送达。',
  'status.copySender': '复制发件邮箱',
  'status.senderCopied': '发件邮箱已复制：',
  'status.copySenderFailed': '复制失败，请手动选择邮箱复制',

  // 卡片编号
  'status.cardCodePrefix': '卡片编号：',
  'status.cardNotGenerated': '尚未生成',
  'status.manualRefresh': '手动刷新',

  // 时间线步骤名
  'status.stepSubmit': '提交申请',
  'status.stepReview': '审核',
  'status.stepMakeCard': '制卡',
  'status.stepMail': '邮寄',
  'status.stepSign': '签收',
  'status.stepDeliver': '交付',
  'status.stepDone': '完成',
  'status.stepSwlYouSend': '您寄出卡',
  'status.stepSwlWeReceive': '我方收卡',
  'status.stepSwlMakeReturn': '制卡回寄',

  // 时间线节点状态说明
  'status.reviewPending': '审核中',
  'status.reviewApproved': '已通过',
  'status.reviewRejected': '未通过',
  'status.cardCreatedPrefix': '已建卡 ',
  'status.cardWaiting': '等待建卡',
  'status.mailed': '已寄出',
  'status.preparing': '准备中',
  'status.waitingMail': '等待',
  'status.signed': '对方已签收',
  'status.waitingSign': '等待签收',
  'status.sentAtPrefix': '已寄出 ',
  'status.addressSent': '地址已发送，请尽快寄出',
  'status.waitingAddress': '等待回寄地址',
  'status.receivedAtPrefix': '已收到 ',
  'status.waitingArrival': '等待寄达',
  'status.delivered': '已交付',
  'status.eyeballDeliverMethods': '见面交付或邮寄',
  'status.waitingCard': '等待制卡',
  'status.completed': '已完成',
  'status.pendingConfirm': '待确认',

  // 申请场景文案
  'status.sceneQso': 'QSO 通联',
  'status.sceneSwl': 'SWL 收听',
  'status.sceneEyeball': 'EYEBALL 见面',
  'status.eyeballOnline': '（网络EYE）',
  'status.eyeballOffline': '（线下补换）'
}
