// set-icon.mjs
// 网站图标源图管理: 约定 pictures/ 恒持有一张"当前源图", 备份统一放 pictures/bak/
// 用法:
//   node scripts/set-icon.mjs <图片路径>   安装新源图, 旧图带时间戳挪入 bak
//   node scripts/set-icon.mjs             查看当前源图与备份列表
import { readdirSync, renameSync, copyFileSync, unlinkSync, mkdirSync, existsSync, statSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, resolve, join, basename, extname } from 'node:path';

const ROOT = resolve(dirname(fileURLToPath(import.meta.url)), '..');
const PIC_DIR = join(ROOT, 'pictures');
const BAK_DIR = join(PIC_DIR, 'bak');
const IMAGE_EXTS = new Set(['.png', '.jpg', '.jpeg', '.webp']);

const isImage = (name) => IMAGE_EXTS.has(extname(name).toLowerCase());

function listImages() {
  return readdirSync(PIC_DIR, { withFileTypes: true })
    .filter((e) => e.isFile() && isImage(e.name))
    .map((e) => e.name)
    .sort();
}

function stamp() {
  const d = new Date();
  const p = (n) => String(n).padStart(2, '0');
  return `${d.getFullYear()}${p(d.getMonth() + 1)}${p(d.getDate())}-${p(d.getHours())}${p(d.getMinutes())}${p(d.getSeconds())}`;
}

// 源图可能来自另一个盘符, rename 抛 EXDEV 时回退为复制+删除
function moveFile(from, to) {
  try {
    renameSync(from, to);
  } catch (err) {
    if (err.code !== 'EXDEV') throw err;
    copyFileSync(from, to);
    unlinkSync(from);
  }
}

function status() {
  const cur = listImages();
  const bak = existsSync(BAK_DIR) ? readdirSync(BAK_DIR).sort() : [];
  console.log('当前源图:', cur.length ? cur.map((n) => `pictures/${n}`).join(', ') : '(无)');
  console.log('备份:', bak.length ? bak.map((n) => `pictures/bak/${n}`).join(', ') : '(空)');
}

const arg = process.argv[2];
if (!arg) {
  status();
  console.log('\n用法: node scripts/set-icon.mjs <新源图路径>');
  process.exit(0);
}

const src = resolve(arg);
if (!existsSync(src) || !statSync(src).isFile()) {
  console.error(`❌ 找不到图片: ${arg}`);
  process.exit(1);
}
if (!isImage(basename(src))) {
  console.error(`❌ 不支持的图片类型: ${extname(src)} (支持 ${[...IMAGE_EXTS].join(' ')})`);
  process.exit(1);
}

// 相对 PIC_DIR 解析, 已在 pictures/ 内的图不重复挪入
const incoming = basename(src);
const insidePic = dirname(src) === PIC_DIR ? incoming : null;

mkdirSync(BAK_DIR, { recursive: true });
const ts = stamp();

// 1) 现存源图全部挪入 bak (时间戳前缀防重名)
for (const name of listImages()) {
  if (name === insidePic) continue;
  moveFile(join(PIC_DIR, name), join(BAK_DIR, `${ts}-${name}`));
  console.log('备份:', `pictures/${name} -> pictures/bak/${ts}-${name}`);
}

// 2) 新图不在 pictures/ 内则挪入
if (!insidePic) {
  moveFile(src, join(PIC_DIR, incoming));
  console.log('安装:', `${arg} -> pictures/${incoming}`);
}

const cur = listImages();
if (cur.length !== 1 || cur[0] !== incoming) {
  console.error(`❌ 异常: pictures/ 内源图不止一张 (${cur.join(', ')}), 请手工检查`);
  process.exit(1);
}
console.log('');
status();
