function generateId(length = 6): string {
	const chars = 'abcdefghijklmnopqrstuvwxyz0123456789';
	let id = '';
	const bytes = crypto.getRandomValues(new Uint8Array(length));
	for (const byte of bytes) {
		id += chars[byte % chars.length];
	}
	return id;
}

export default {
	async fetch(request, env, ctx): Promise<Response> {
		const url = new URL(request.url);
		const { pathname } = url;

		// Simple upload for small files (< 100MB)
		if (pathname === '/upload' && request.method === 'POST') {
			const action = url.searchParams.get('action');

			// Multipart: create upload
			if (action === 'mpu-create') {
				const filename = generateId();
				const mpu = await env.BUCKET.createMultipartUpload(filename);
				return Response.json({
					filename: mpu.key,
					uploadId: mpu.uploadId,
				});
			}

			// Multipart: complete upload
			if (action === 'mpu-complete') {
				const filename = url.searchParams.get('filename');
				const uploadId = url.searchParams.get('uploadId');
				if (!filename || !uploadId) {
					return Response.json({ error: 'Missing filename or uploadId' }, { status: 400 });
				}
				const mpu = env.BUCKET.resumeMultipartUpload(filename, uploadId);
				const body: { parts: R2UploadedPart[] } = await request.json();
				try {
					await mpu.complete(body.parts);
					return Response.json({ filename });
				} catch (err: any) {
					return Response.json({ error: err.message }, { status: 400 });
				}
			}

			// Simple single-request upload
			if (!action) {
				const body = request.body;
				if (!body) {
					return Response.json({ error: 'No file provided' }, { status: 400 });
				}
				const contentType = request.headers.get('content-type') || 'application/octet-stream';
				const filename = generateId();
				await env.BUCKET.put(filename, body, {
					httpMetadata: { contentType },
				});
				return Response.json({ filename });
			}

			return Response.json({ error: `Unknown action: ${action}` }, { status: 400 });
		}

		// Multipart: upload a part
		if (pathname === '/upload' && request.method === 'PUT') {
			const filename = url.searchParams.get('filename');
			const uploadId = url.searchParams.get('uploadId');
			const partNumber = url.searchParams.get('partNumber');
			if (!filename || !uploadId || !partNumber) {
				return Response.json({ error: 'Missing filename, uploadId, or partNumber' }, { status: 400 });
			}
			if (!request.body) {
				return Response.json({ error: 'Missing request body' }, { status: 400 });
			}
			const mpu = env.BUCKET.resumeMultipartUpload(filename, uploadId);
			try {
				const part = await mpu.uploadPart(parseInt(partNumber), request.body);
				return Response.json(part);
			} catch (err: any) {
				return Response.json({ error: err.message }, { status: 400 });
			}
		}

		// Multipart: abort upload
		if (pathname === '/upload' && request.method === 'DELETE') {
			const filename = url.searchParams.get('filename');
			const uploadId = url.searchParams.get('uploadId');
			if (!filename || !uploadId) {
				return Response.json({ error: 'Missing filename or uploadId' }, { status: 400 });
			}
			const mpu = env.BUCKET.resumeMultipartUpload(filename, uploadId);
			try {
				await mpu.abort();
			} catch (err: any) {
				return Response.json({ error: err.message }, { status: 400 });
			}
			return new Response(null, { status: 204 });
		}

		// Download a file by filename
		if (pathname === '/download' && request.method === 'GET') {
			const filename = url.searchParams.get('filename');
			if (!filename) {
				return Response.json({ error: 'Missing filename' }, { status: 400 });
			}
			const object = await env.BUCKET.get(filename);
			if (!object) {
				return Response.json({ error: 'Not found' }, { status: 404 });
			}
			const headers = new Headers();
			object.writeHttpMetadata(headers);
			headers.set('etag', object.httpEtag);
			return new Response(object.body, { headers });
		}

		// Delete a file by filename
		if (pathname === '/download' && request.method === 'DELETE') {
			const filename = url.searchParams.get('filename');
			if (!filename) {
				return Response.json({ error: 'Missing filename' }, { status: 400 });
			}
			await env.BUCKET.delete(filename);
			return new Response(null, { status: 204 });
		}

		// Serve unpack script (downloads and extracts to cwd)
		if (pathname === '/unpack' && request.method === 'GET') {
			return Response.redirect(
				'https://raw.githubusercontent.com/damiant/packageup-cli/main/unpack.sh',
				302
			);
		}

		// Serve pack script (archives cwd and uploads)
		if (pathname === '/pack' && request.method === 'GET') {
			return Response.redirect(
				'https://raw.githubusercontent.com/damiant/packageup-cli/main/pack.sh',
				302
			);
		}

		// Redirect to install script
		if (pathname === '/install' && request.method === 'GET') {
			return Response.redirect(
				'https://raw.githubusercontent.com/damiant/packageup-cli/main/install.sh',
				302
			);
		}

		return new Response('Not Found', { status: 404 });
	},
} satisfies ExportedHandler<Env>;
