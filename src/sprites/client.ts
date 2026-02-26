import { SpritesClient, Sprite } from "@fly/sprites";

export class Client {
  private sdk: SpritesClient;
  private org: string | undefined;

  constructor(token: string, org: string) {
    const trimmed = org.trim();
    this.org = trimmed || undefined;
    this.sdk = new SpritesClient(token);
  }

  async createSprite(name: string): Promise<Sprite> {
    const sp = await this.sdk.createSprite(name);
    return this.sdk.getSprite(sp.name);
  }

  async deleteSprite(name: string): Promise<void> {
    await this.sdk.deleteSprite(name);
  }

  async listSpriteNames(): Promise<string[]> {
    const all = await this.sdk.listAllSprites();
    return all.map((sp) => sp.name);
  }

  sprite(name: string): Sprite {
    return this.sdk.sprite(name);
  }

  get organization(): string | undefined {
    return this.org;
  }
}

export function isNameCollision(err: unknown): boolean {
  if (err == null) return false;
  const s = String(err).toLowerCase();
  return s.includes("already exists") || s.includes("name is taken") || s.includes("409");
}
