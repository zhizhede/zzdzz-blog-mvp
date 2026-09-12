// ICP 备案信息从 web/beian.config.json 读取(gitignored, 含真实姓名等个人信息, 不入仓库).
// 用 import.meta.glob 而非静态 import: 文件不存在时构建不报错, 页脚自动隐藏备案链接.
export interface BeianInfo {
  /** 服务域名, 如 zzdzz.cn */
  domain: string
  /** 备案号, 如 粤ICP备xxxxxxxx号-x */
  icp: string
  /** 服务负责人 */
  owner: string
  /** 备案号落地链接, 工信部要求指向 https://beian.miit.gov.cn */
  miit_url: string
}

const mods = import.meta.glob<{ default: BeianInfo }>('../beian.config.json', {
  eager: true,
})

export const beian: BeianInfo | undefined = mods['../beian.config.json']?.default
