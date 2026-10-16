// 平移的 ES 视觉模块（.js）的类型声明面。
declare module '*/visual-assets.js' {
  export function prepareVisualAssets(): Promise<void>;
  export function createModel(name: string, teamColor?: string): any;
  export function weatherMaterial(material: any, domain?: string): any;
  export function modelThumbnail(name: string, color?: string): any;
  export function spriteTexture(name: string, teamColor?: string): any;
  export function visualLibrary(): any;
}
