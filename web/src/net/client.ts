// net/client.ts：WS 客户端——对齐 online-client.js 语义（重连/快照/指令 seq）。
export interface RoomMember {
  id: string; name: string; faction: string; ready: boolean;
  connected: boolean; controlType: string; eliminated?: boolean;
}
export interface PublicRoom {
  code: string | null; host: number; config: any; status: string;
  reason: string; deadline: number | null; members: (RoomMember | null)[];
}
export interface User { id: string; name: string; wins: number; losses: number; draws: number; }

export type ServerMsg =
  | { type: 'hello'; protocol: number; user: User; room: PublicRoom | null }
  | { type: 'room'; room: PublicRoom | null }
  | { type: 'battle' | 'snapshot'; config: any; seat: number; room: PublicRoom; spectator: boolean; view: any }
  | { type: 'ack'; seq: number; ok: boolean; action: string }
  | { type: 'pong'; at: any }
  | { type: 'error'; message: string };

export class NetClient {
  ws: WebSocket | null = null;
  user: User | null = null;
  room: PublicRoom | null = null;
  view: any = null;
  seat = 0;
  private seq = 0;
  private handlers = new Set<(m: ServerMsg) => void>();
  private retryTimer: number | null = null;

  onMessage(h: (m: ServerMsg) => void) {
    this.handlers.add(h);
    return () => this.handlers.delete(h);
  }
  private emit(m: ServerMsg) { this.handlers.forEach(h => h(m)); }

  async session(): Promise<void> {
    let res = await fetch('/api/session');
    let data = await res.json();
    if (!data.user) {
      // 无 cookie 首访：建匿名号（服务器下 Set-Cookie），再回读。
      res = await fetch('/api/session', {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: '指挥官' }),
      });
      data = await res.json();
    }
    this.user = data.user ?? null;
    this.room = data.room ?? null;
  }

  async rename(name: string): Promise<void> {
    const res = await fetch('/api/session', {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name }),
    });
    const data = await res.json();
    if (data.user) this.user = data.user;
  }

  connect(url = `ws://${location.host}/battle-socket`): Promise<void> {
    return new Promise<void>(resolve => {
      const ws = new WebSocket(url);
      this.ws = ws;
      const done = () => resolve();
      if (ws.readyState === WebSocket.OPEN) done();
      else ws.addEventListener('open', done, { once: true });
      // 首连失败（如 cookie 未就绪）时短退避重连，直至 open。
      ws.addEventListener('close', () => {
        if (this.ws !== ws) return;
        this.retryTimer = window.setTimeout(() => this.connect(url).then(resolve), 300);
      }, { once: true });
    }).then(() => this.listen());
  }

  private listen() {
    const ws = this.ws!;
    ws.onmessage = ev => {
      const m = JSON.parse(ev.data) as ServerMsg;
      if (m.type === 'hello') { this.user = m.user; this.room = m.room; }
      if (m.type === 'room') this.room = m.room;
      if (m.type === 'battle' || m.type === 'snapshot') { this.view = m.view; this.seat = m.seat; }
      this.emit(m);
    };
    ws.onclose = () => {
      this.retryTimer = window.setTimeout(() => { this.connect(); }, 2000);
    };
  }

  get connected() { return this.ws?.readyState === WebSocket.OPEN; }

  send(msg: Record<string, unknown>) {
    if (this.ws?.readyState === WebSocket.OPEN) this.ws.send(JSON.stringify(msg));
  }
  command(action: string, extra: Record<string, unknown> = {}) {
    this.seq++;
    this.send({ type: 'command', seq: this.seq, action, ...extra });
  }
  create(mapId: string, faction: string, victoryMode: string) {
    this.send({ type: 'create', mapId, faction, victoryMode });
  }
  /** 单机：1 号席位自动由 AI 接管，服务端开局即 Multiplayer=false。 */
  createSolo(mapId: string, faction: string, victoryMode: string) {
    this.send({ type: 'create', mapId, faction, victoryMode, solo: true });
  }
  join(code: string, faction: string) { this.send({ type: 'join', code, faction }); }
  ready(ready: boolean) { this.send({ type: 'ready', ready }); }
  faction(faction: string) { this.send({ type: 'faction', faction }); }
  start() { this.send({ type: 'start' }); }
  leave() { this.send({ type: 'leave' }); }
  ping(at: number) { this.send({ type: 'ping', at }); }
  close() {
    if (this.retryTimer) clearTimeout(this.retryTimer);
    this.ws?.close();
  }
}
