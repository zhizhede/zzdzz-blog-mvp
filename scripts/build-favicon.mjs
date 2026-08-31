// build-favicon.mjs
// 从 pictures/zzdzz-blog-page.png 生成 favicon 各尺寸 (SVG/PNG/ICO)
import sharp from 'sharp';
import { readFileSync, writeFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';

const __dirname = dirname(fileURLToPath(import.meta.url));
const ROOT = resolve(__dirname, '..');
const SRC = resolve(ROOT, 'pictures/zzdzz-blog-page.png');

// 原图 1395x1395，直接整体使用，不裁剪
const CROP = {
  left: 0,
  top: 0,
  width: 1395,
  height: 1395,
};

const SIZES = [32, 48, 64, 128, 256, 512];

async function build() {
  const srcBuf = readFileSync(SRC);
  const meta = await sharp(srcBuf).metadata();
  console.log('source:', meta.width, 'x', meta.height);

  // 1) 生成中间方形 PNG（512px，作为裁剪后的源）
  const square512 = await sharp(srcBuf)
    .extract(CROP)
    .resize(512, 512, { fit: 'cover' })
    .png()
    .toBuffer();

  // 2) 写 favicon.svg = 单 PNG 转 base64 内嵌（保持矢量占位 + 位图精度）
  //    实际上浏览器对 SVG favicon 支持有限但能渲染位图嵌入，更稳的是写 PNG/ICO。
  //    这里仍然输出 SVG 但用 PNG 内嵌，保持单一来源。
  const png256 = await sharp(square512).resize(256, 256).png().toBuffer();
  const b64 = png256.toString('base64');
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 256 256" width="256" height="256">
  <image href="data:image/png;base64,${b64}" width="256" height="256"/>
</svg>\n`;

  const targets = [
    resolve(ROOT, 'web/public/favicon.svg'),
    resolve(ROOT, 'web/demo/public/favicon.svg'),
    resolve(ROOT, 'web/dist/favicon.svg'),
  ];
  for (const t of targets) {
    writeFileSync(t, svg);
    console.log('wrote svg ->', t);
  }

  // 3) 各尺寸 PNG
  for (const size of SIZES) {
    const out = await sharp(square512)
      .resize(size, size)
      .png()
      .toBuffer();
    for (const dir of ['web/public', 'web/demo/public']) {
      const path = resolve(ROOT, dir, `favicon-${size}.png`);
      writeFileSync(path, out);
    }
    console.log('wrote png', size);
  }

  // 4) favicon.ico (16+32+48)
  const icoBuffers = await Promise.all([16, 32, 48].map(async (s) => {
    const buf = await sharp(square512).resize(s, s).png().toBuffer();
    return { size: s, buf };
  }));

  // 简易 ICO 容器构造（PNG 内嵌格式）
  const icoHeader = Buffer.alloc(6);
  icoHeader.writeUInt16LE(0, 0);  // reserved
  icoHeader.writeUInt16LE(1, 2);  // type: ICO
  icoHeader.writeUInt16LE(icoBuffers.length, 4);

  const dirEntries = [];
  let offset = 6 + 16 * icoBuffers.length;
  for (const { size, buf } of icoBuffers) {
    const dim = size >= 256 ? 0 : size;
    const entry = Buffer.alloc(16);
    entry.writeUInt8(dim, 0);
    entry.writeUInt8(dim, 1);
    entry.writeUInt8(0, 2); // colors
    entry.writeUInt8(0, 3); // reserved
    entry.writeUInt16LE(1, 4); // planes
    entry.writeUInt16LE(32, 6); // bpp
    entry.writeUInt32LE(buf.length, 8);
    entry.writeUInt32LE(offset, 12);
    dirEntries.push(entry);
    offset += buf.length;
  }

  const ico = Buffer.concat([icoHeader, ...dirEntries, ...icoBuffers.map((b) => b.buf)]);
  for (const dir of ['web/public', 'web/demo/public']) {
    const path = resolve(ROOT, dir, 'favicon.ico');
    writeFileSync(path, ico);
  }
  console.log('wrote ico');

  // 5) apple-touch-icon (180x180)
  const apple = await sharp(square512).resize(180, 180).png().toBuffer();
  for (const dir of ['web/public', 'web/demo/public']) {
    writeFileSync(resolve(ROOT, dir, 'apple-touch-icon.png'), apple);
  }
  console.log('wrote apple-touch-icon');
}

build().catch((e) => { console.error(e); process.exit(1); });
