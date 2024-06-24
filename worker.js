var ACCESS_TOKEN, REDIRECT_URL = 'http://127.0.0.1:8787/callback'
const API_URL = 'https://cloudshell.googleapis.com/v1/users/me/environments/default'
const SCOPE = 'https://www.googleapis.com/auth/cloud-platform'
const ERROR = {
	'UNAUTHENTICATED': {
		error: {
			code: 401,
			status: 'UNAUTHENTICATED'
		}
	},
	'UNKNOWN': {
		error: {
			code: 500,
			status: 'ERROR'
		}
	}
}

async function getStatus() {
	const response = await fetch(API_URL, {
		headers: {
			'Authorization': `Bearer ${ACCESS_TOKEN}`
		}
	})

	let data = await response.json()

	if (data.error) {
		return {
			'error': data.error
		}
	} else if (data.state == 'RUNNING') {
		return {
			'state': data.state,
			'sshUsername': data.sshUsername,
			'sshPort': data.sshPort,
			'sshHost': data.sshHost
		}
	} else {
		return {
			'state': data.state
		}
	}
}

async function start() {
	const response = await fetch(API_URL + ':start', {
		method: 'POST',
		headers: {
			'Authorization': `Bearer ${ACCESS_TOKEN}`,
			"content-type": "application/json;charset=UTF-8",
		},
		body: JSON.stringify({
			"accessToken": ACCESS_TOKEN,
			"publicKeys": []
		})
	})

	return response.json()
}

async function addPublicKey(env) {
	const response = await fetch(API_URL + ':addPublicKey', {
		method: 'POST',
		headers: {
			'Authorization': `Bearer ${ACCESS_TOKEN}`,
			"content-type": "application/json;charset=UTF-8",
		},
		body: JSON.stringify({
			"key": env.SSH_PUBLICKEY
		})
	})

	return response.json()
}

async function getAccessTokenKey(request, env) {
	// const ClientIP = request.headers.get('CF-Connecting-IP')
	const ClientIP = ''
	return btoa(ClientIP + env.SECRET_KEY).replaceAll('=', '')
}

async function getAccessToken(request, env) {
	// let token = await env.KV.get(key)
	const key = await getAccessTokenKey(request, env)
	const time = new Date().getTime()
	const ps = env.DB.prepare('SELECT value FROM kv WHERE key = ? and expires > ? LIMIT 1').bind(key, time)
	const data = await ps.first('value')

	// console.log(data)
	return data
}

async function putAccessToken(request, env, data) {
	// await env.KV.put(key, data.access_token, { expirationTtl: data.expires_in })
	const key = await getAccessTokenKey(request, env)
	const expires = (data.expires_in * 1000) + new Date().getTime()
	await env.DB.prepare('DELETE FROM kv WHERE key = ?').bind(key).run()
	const info = await env.DB.prepare('INSERT INTO kv (key, value, expires) VALUES (?1, ?2, ?3)')
		.bind(key, data.access_token, expires)
		.run()

	console.log(info)
}

async function handleCallback(request, env) {
	const tokenUrl = 'https://www.googleapis.com/oauth2/v4/token';

	const url = new URL(request.url)
	const code = url.searchParams.get('code')

	const response = await fetch(tokenUrl, {
		method: 'POST',
		headers: {
			'Content-Type': 'application/x-www-form-urlencoded'
		},
		body: `code=${encodeURIComponent(code)}&client_id=${env.CLIENT_ID}&client_secret=${env.CLIENT_SECRET}&redirect_uri=${encodeURIComponent(REDIRECT_URL)}&grant_type=authorization_code`
	})

	const data = await response.json()

	if (data.access_token) {
		await putAccessToken(request, env, data)
		// console.log(data)
		return new Response('Authorization successful. You can now close the page.')
	} else {
		return Response.json(data)
	}
}

function sleep(ms) {
	return new Promise(resolve => setTimeout(resolve, ms));
}

async function handleToken(request, env) {
	if (ACCESS_TOKEN == "") {
		return Response.json({ "state": "false" })
	}
	return Response.json({ "state": "true" })
}

async function handleStatus(request, env) {
	const data = await getStatus()
	// console.log(data)
	if (data.error) {
		return Response.json(data)
	} else if (data.state == 'RUNNING') {
		return Response.json(data)
	} else if (data.state == 'SUSPENDED') {
		return Response.json({ "state": data.state })
	} else {
		return Response.json(data)
	}
}

async function handleStart(request, env) {
	const data = start()
	return Response.json(data)
}

async function handleConnect(request, env) {
	let response = await getStatus()

	if (response.error) {
		return Response.json(response)
	} else if (response.state == 'SUSPENDED') {
		response = await start()

		if (response.error) {
			return Response.json({
				'error': response.error
			})
		} else if (response.metadata && response.metadata.state == 'STARTING') {
			// for (let i = 0; i < 3; i++) {
			// 	await sleep(1000)
			// 	response = await getStatus()
			// 	if (response.state == 'RUNNING') {
			// 		break;
			// 	}
			// }

			return Response.json(response)
		} else {
			console.log(response)
			return Response.json(ERROR.UNKNOWN)
		}
	} else if (response.state == 'RUNNING') {
		return Response.json(response)
	} else if (response.state == 'STARTING') {
		return Response.json(response)
	} else {
		console.log(response)
		return Response.json(ERROR.UNKNOWN)
	}
}

export default {
	async fetch(request, env, ctx) {
		const url = new URL(request.url);
		REDIRECT_URL = url.origin + '/callback'

		if (url.pathname === '/callback') {
			return handleCallback(request, env)
		} else if (url.pathname === '/auth') {
			const url = `https://accounts.google.com/o/oauth2/v2/auth?client_id=${env.CLIENT_ID}&redirect_uri=${REDIRECT_URL}&scope=${encodeURIComponent(SCOPE)}&response_type=code`
			return Response.redirect(url);
		} else {
			ACCESS_TOKEN = await getAccessToken(request, env)
			if (!ACCESS_TOKEN) {
				return Response.json(ERROR.UNAUTHENTICATED)
			}

			if (url.pathname === '/token') {
				// return handleToken(request, env)
			} else if (url.pathname === '/status') {
				return handleStatus(request, env)
			} else if (url.pathname === '/start') {
				// return handleStart(request, env)
			} else if (url.pathname === '/connect') {
				return handleConnect(request, env)
			} if (url.pathname === '/addPublicKey') {
				// let response = await addPublicKey(env)
				// return Response.json(response)
			}
		}

		// handleRequest
		return new Response("Hello, World!")
	}
}
