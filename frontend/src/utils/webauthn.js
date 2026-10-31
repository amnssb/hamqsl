/*
浏览器通行密钥胶水。

服务器下发的是 JSON（base64url 字符串），navigator.credentials 要的是 ArrayBuffer；
浏览器返回的凭据又得转回 JSON 才能发给服务器。两头都由这里做转换。

调用示例：
	const opts = publicKeyToCreationOptions(res.publicKey)
	const cred = await navigator.credentials.create({ publicKey: opts })
	await api.post('/auth/passkey/register/finish?ticket=' + res.ticket, credentialToJSON(cred))
*/

// base64url → ArrayBuffer。WebAuthn 全程用 base64url，不能直接 atob 标准 base64。
export function b64urlToBuffer(value) {
	const pad = value.length % 4 === 0 ? '' : '='.repeat(4 - (value.length % 4))
	const b64 = (value + pad).replace(/-/g, '+').replace(/_/g, '/')
	const bin = atob(b64)
	const bytes = new Uint8Array(bin.length)
	for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i)
	return bytes.buffer
}

// ArrayBuffer → base64url（去掉 = 填充，和服务器保持一致）
export function bufferToB64url(buf) {
	const bytes = new Uint8Array(buf)
	let bin = ''
	for (let i = 0; i < bytes.length; i++) bin += String.fromCharCode(bytes[i])
	return btoa(bin).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
}

// 描述符里的 id 也要解码（注册时的 excludeCredentials、登录时的 allowCredentials）
function decodeIds(list) {
	return (list || []).map((item) => ({ ...item, id: b64urlToBuffer(item.id) }))
}

// 注册选项：challenge 与 user.id 都是二进制
export function publicKeyToCreationOptions(pk) {
	return {
		...pk,
		challenge: b64urlToBuffer(pk.challenge),
		user: { ...pk.user, id: b64urlToBuffer(pk.user.id) },
		excludeCredentials: decodeIds(pk.excludeCredentials),
	}
}

// 登录/验证选项
export function publicKeyToRequestOptions(pk) {
	return {
		...pk,
		challenge: b64urlToBuffer(pk.challenge),
		allowCredentials: decodeIds(pk.allowCredentials),
	}
}

// 浏览器凭据 → 普通对象。注册（attestationObject）和断言（signature）字段不同，按存在与否分开取。
export function credentialToJSON(cred) {
	const out = {
		id: cred.id,
		rawId: bufferToB64url(cred.rawId),
		type: cred.type,
		clientExtensionResults: cred.getClientExtensionResults ? cred.getClientExtensionResults() : {},
		authenticatorAttachment: cred.authenticatorAttachment || null,
	}

	if (cred.response.attestationObject) {
		out.response = {
			clientDataJSON: bufferToB64url(cred.response.clientDataJSON),
			attestationObject: bufferToB64url(cred.response.attestationObject),
		}
		return out
	}

	out.response = {
		clientDataJSON: bufferToB64url(cred.response.clientDataJSON),
		authenticatorData: bufferToB64url(cred.response.authenticatorData),
		signature: bufferToB64url(cred.response.signature),
		userHandle: cred.response.userHandle ? bufferToB64url(cred.response.userHandle) : null,
	}
	return out
}

// 当前环境能不能用通行密钥。
// 明文 HTTP（或者端口不常规的 http 地址）不是安全上下文，浏览器直接不提供 navigator.credentials，
// 所以按钮要按这个结果藏起来，而不是让用户点了报错。
export function passkeySupported() {
	return !!(window.isSecureContext && window.PublicKeyCredential && navigator.credentials)
}
