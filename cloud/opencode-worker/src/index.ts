export interface Env {
  SANDBOX_DO: DurableObjectNamespace;
  ENVIRONMENT: string;
  OPENCODE_AUTH_OPENAI_B64?: string;
}

export class Sandbox implements DurableObject {
  private state: DurableObjectState;
  private env: Env;
  private status: 'pending' | 'running' | 'stopped' | 'error' = 'pending';
  private createdAt: number = 0;
  private openUrl: string = '';
  private errorMessage: string = '';

  constructor(state: DurableObjectState, env: Env) {
    this.state = state;
    this.env = env;
  }

  async fetch(request: Request): Promise<Response> {
    const url = new URL(request.url);
    const path = url.pathname;

    if (path === '/status' && request.method === 'GET') {
      return Response.json(this.getStatus());
    }

    if (path === '/launch' && request.method === 'POST') {
      return this.handleLaunch(request);
    }

    if (path === '/teardown' && request.method === 'POST') {
      return this.handleTeardown();
    }

    if (path === '/open-url' && request.method === 'GET') {
      return this.handleOpenUrl();
    }

    if (path === '/proxy' && request.method === 'GET') {
      return this.handleProxy(request);
    }

    return Response.json({ error: 'not_found', reason: 'endpoint_not_found' }, { status: 404 });
  }

  private getStatus() {
    return {
      status: this.status,
      created_at: this.createdAt,
      open_url: this.openUrl,
      error: this.errorMessage || undefined,
    };
  }

  private validateAuth(): void {
    if (!this.env.OPENCODE_AUTH_OPENAI_B64) {
      throw new Error(
        'OPENCODE_AUTH_OPENAI_B64 is not set. ' +
        'Generate it using: scripts/sync-opencode-auth.sh --local'
      );
    }
    
    try {
      const decoded = atob(this.env.OPENCODE_AUTH_OPENAI_B64);
      JSON.parse(decoded);
    } catch {
      throw new Error(
        'OPENCODE_AUTH_OPENAI_B64 is not valid base64-encoded JSON. ' +
        'Regenerate using: scripts/sync-opencode-auth.sh --local'
      );
    }
  }

  private async handleLaunch(request: Request): Promise<Response> {
    if (this.status === 'running') {
      return Response.json({
        ok: true,
        reason: 'already_running',
        status: this.getStatus(),
      });
    }

    try {
      this.validateAuth();
      
      const body = await request.json<{ name?: string }>().catch(() => ({ name: undefined }));
      const name = body.name || 'default';
      
      this.createdAt = Date.now();
      this.status = 'running';
      this.openUrl = `/sandbox/${name}/ui`;
      this.errorMessage = '';

      return Response.json({
        ok: true,
        reason: 'launched',
        status: this.getStatus(),
      });
    } catch (err) {
      this.status = 'error';
      this.errorMessage = err instanceof Error ? err.message : 'unknown_error';
      return Response.json(
        { ok: false, reason: 'launch_failed', error: this.errorMessage },
        { status: 500 }
      );
    }
  }

  private async handleTeardown(): Promise<Response> {
    if (this.status !== 'running') {
      return Response.json({
        ok: true,
        reason: 'not_running',
        status: this.getStatus(),
      });
    }

    this.status = 'stopped';
    this.openUrl = '';

    return Response.json({
      ok: true,
      reason: 'teardown_complete',
      status: this.getStatus(),
    });
  }

  private async handleOpenUrl(): Promise<Response> {
    if (this.status !== 'running') {
      return Response.json(
        { ok: false, reason: 'not_running', error: 'sandbox is not running' },
        { status: 400 }
      );
    }

    return Response.json({
      ok: true,
      reason: 'success',
      open_url: this.openUrl,
    });
  }

  private async handleProxy(request: Request): Promise<Response> {
    if (this.status !== 'running') {
      return Response.json(
        { ok: false, reason: 'not_running', error: 'sandbox is not running' },
        { status: 503 }
      );
    }

    return new Response('OpenCode UI placeholder - proxy to sandbox container', {
      headers: { 'content-type': 'text/plain' },
    });
  }
}

function jsonResponse(data: unknown, status = 200): Response {
  return Response.json(data, { status });
}

function errorResponse(reason: string, error: string, status = 400): Response {
  return Response.json({ ok: false, reason, error }, { status });
}

async function getSandboxDO(env: Env, name: string): Promise<DurableObjectStub> {
  const id = env.SANDBOX_DO.idFromName(name);
  return env.SANDBOX_DO.get(id);
}

async function handleControlPlane(request: Request, env: Env): Promise<Response> {
  const url = new URL(request.url);
  const path = url.pathname;

  if (path === '/healthz' && request.method === 'GET') {
    return jsonResponse({ ok: true, reason: 'healthy', environment: env.ENVIRONMENT });
  }

  const sandboxesMatch = path.match(/^\/v1\/sandboxes$/);
  if (sandboxesMatch && request.method === 'GET') {
    return jsonResponse({ ok: true, sandboxes: [] });
  }

  const launchMatch = path.match(/^\/v1\/sandboxes\/([^/]+)\/launch$/);
  if (launchMatch && request.method === 'POST') {
    const name = launchMatch[1];
    const stub = await getSandboxDO(env, name);
    return stub.fetch(new Request('http://internal/launch', {
      method: 'POST',
      headers: { 'content-type': 'application/json' },
      body: JSON.stringify({ name }),
    }));
  }

  const statusMatch = path.match(/^\/v1\/sandboxes\/([^/]+)\/status$/);
  if (statusMatch && request.method === 'GET') {
    const name = statusMatch[1];
    const stub = await getSandboxDO(env, name);
    return stub.fetch(new Request('http://internal/status', { method: 'GET' }));
  }

  const openUrlMatch = path.match(/^\/v1\/sandboxes\/([^/]+)\/open-url$/);
  if (openUrlMatch && request.method === 'GET') {
    const name = openUrlMatch[1];
    const stub = await getSandboxDO(env, name);
    return stub.fetch(new Request('http://internal/open-url', { method: 'GET' }));
  }

  const teardownMatch = path.match(/^\/v1\/sandboxes\/([^/]+)\/teardown$/);
  if (teardownMatch && request.method === 'POST') {
    const name = teardownMatch[1];
    const stub = await getSandboxDO(env, name);
    return stub.fetch(new Request('http://internal/teardown', { method: 'POST' }));
  }

  if (path.startsWith('/sandbox/') && path.includes('/ui')) {
    const nameMatch = path.match(/^\/sandbox\/([^/]+)\/ui/);
    if (nameMatch) {
      const name = nameMatch[1];
      const stub = await getSandboxDO(env, name);
      return stub.fetch(new Request('http://internal/proxy' + url.search, { method: 'GET' }));
    }
  }

  return errorResponse('not_found', 'endpoint not found', 404);
}

export default {
  async fetch(request: Request, env: Env, _ctx: ExecutionContext): Promise<Response> {
    try {
      return await handleControlPlane(request, env);
    } catch (err) {
      console.error('worker error:', err);
      return errorResponse(
        'internal_error',
        err instanceof Error ? err.message : 'unknown error',
        500
      );
    }
  },
};
