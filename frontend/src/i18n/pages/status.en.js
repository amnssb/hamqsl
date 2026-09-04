// Request status page strings (to be filled during RequestStatus.vue i18n pass)
export default {
  // Header & lookup entry
  'status.title': 'Application Status',
  'status.subtitle': 'Enter your request code to view review, card making, shipping and delivery status',
  'status.requestCode': 'Request Code',
  'status.requestCodePlaceholder': 'e.g. EXAB23CD9',
  'status.lookupBtn': 'Check Status',
  'status.loading': 'Querying application status...',
  'status.failedTitle': 'Query Failed',
  'status.reenter': 'Enter Again',
  'status.notFound': 'Application does not exist or the code is wrong',

  // Detail base fields
  'status.callSign': 'Call Sign',
  'status.scene': 'Scenario',
  'status.requestTime': 'Requested At',
  'status.notice': 'Notice',

  // Review result
  'status.rejectedPrefix': 'Application rejected: ',
  'status.noReason': 'No reason given',

  // Parcel tracking
  'status.mailTracking': 'Parcel Tracking',
  'status.kuaidiTip': 'Copy the tracking number below and check real-time delivery on the Kuaidi100 website:',
  'status.openKuaidi': 'Track on Kuaidi100',
  'status.openKuaidiShort': 'Check on Kuaidi100',
  'status.copyTracking': 'Copy Tracking No.',
  'status.trackingCopied': 'Tracking number copied',
  'status.copyTrackingFailed': 'Copy failed. Please select the tracking number and copy it manually',

  // SWL return mail
  'status.sendToAddress': 'Please mail your card to the address below',
  'status.registerMailHead': 'Register Your Outgoing Mail',
  'status.registerMailTip': 'After you mail your listening card, register it here; we will log your shipment and process the return mail as soon as possible.',
  'status.mailType': 'Mail Type',
  'status.registeredMail': 'Registered Mail',
  'status.ordinaryMail': 'Ordinary Mail',
  'status.trackingNo': 'Tracking Number',
  'status.trackingPlaceholder': 'Your registered mail or courier tracking number',
  'status.submitRegister': 'Submit Registration',
  'status.trackingRequired': 'Registered mail requires a tracking number',
  'status.registerSuccess': 'Registration successful. We have received your outgoing mail record',
  'status.registeredHead': 'Your Outgoing Mail Registration',
  'status.trackingPrefix': 'Tracking No.: ',
  'status.registeredAtPrefix': 'Registered at: ',

  // Whitelist guidance
  'status.mailNotReceived': 'Email Not Received?',
  'status.whitelistTipBefore': 'Email notifications are sent at key stages: review approval, card receipt, and return card shipment. If you have not received them, please check your Spam folder and add the sender address ',
  'status.whitelistTipAfter': ' to your whitelist so future notifications reach your inbox.',
  'status.copySender': 'Copy Sender Email',
  'status.senderCopied': 'Sender email copied: ',
  'status.copySenderFailed': 'Copy failed. Please select the email address and copy it manually',

  // Card code
  'status.cardCodePrefix': 'Card Code: ',
  'status.cardNotGenerated': 'Not generated yet',
  'status.manualRefresh': 'Refresh',

  // Timeline step titles
  'status.stepSubmit': 'Application Submitted',
  'status.stepReview': 'Review',
  'status.stepMakeCard': 'Card Making',
  'status.stepMail': 'Shipping',
  'status.stepSign': 'Delivered',
  'status.stepDeliver': 'Handover',
  'status.stepDone': 'Completed',
  'status.stepSwlYouSend': 'You Mail Your Card',
  'status.stepSwlWeReceive': 'We Receive Your Card',
  'status.stepSwlMakeReturn': 'Card Making & Return Mail',

  // Timeline node status descriptions
  'status.reviewPending': 'Under Review',
  'status.reviewApproved': 'Approved',
  'status.reviewRejected': 'Rejected',
  'status.cardCreatedPrefix': 'Card created: ',
  'status.cardWaiting': 'Waiting for card making',
  'status.mailed': 'Shipped',
  'status.preparing': 'Preparing',
  'status.waitingMail': 'Waiting',
  'status.signed': 'Signed for by recipient',
  'status.waitingSign': 'Awaiting delivery',
  'status.sentAtPrefix': 'Sent at ',
  'status.addressSent': 'Address sent. Please mail your card soon',
  'status.waitingAddress': 'Waiting for return address',
  'status.receivedAtPrefix': 'Received at ',
  'status.waitingArrival': 'Awaiting arrival',
  'status.delivered': 'Delivered',
  'status.eyeballDeliverMethods': 'In-person handover or by mail',
  'status.waitingCard': 'Waiting for card making',
  'status.completed': 'Completed',
  'status.pendingConfirm': 'Awaiting confirmation',

  // Scenario labels
  'status.sceneQso': 'QSO Contact',
  'status.sceneSwl': 'SWL Listening',
  'status.sceneEyeball': 'EYEBALL Meetup',
  'status.eyeballOnline': ' (Online EYE)',
  'status.eyeballOffline': ' (Offline Replacement)'
}
