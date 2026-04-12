'use client'

import { type ReactNode, useMemo, useState } from 'react'

type CategoryKey = 'basic' | 'security' | 'notifications' | 'storage' | 'integrations' | 'advanced'

type SettingsState = {
  siteName: string
  siteUrl: string
  siteDescription: string
  adminEmail: string
  defaultLanguage: string
  timezone: string
  dateFormat: string
  allowRegistration: boolean
  emailNotifications: boolean
  notificationSender: string
  smtpHost: string
  smtpPort: number
  supportEmail: string
  maintenanceMode: boolean
  forceHttps: boolean
  twoFactorAuth: boolean
  passwordMinLength: number
  sessionTimeout: number
  maxLoginAttempts: number
  auditRetentionDays: number
  storageProvider: string
  storageBucket: string
  retentionDays: number
  integrationWebhook: string
  ssoProvider: string
  featureFlags: {
    enableAuditStreaming: boolean
    enablePortal: boolean
  }
}

const defaultSettings = (): SettingsState => ({
  siteName: 'PowerX Plugin IAM',
  siteUrl: 'https://plugin.localhost',
  siteDescription: '集中管理租户、部门与角色的 Standalone 控制台',
  adminEmail: 'admin@example.com',
  defaultLanguage: 'zh-CN',
  timezone: 'Asia/Shanghai',
  dateFormat: 'YYYY-MM-DD',
  allowRegistration: true,
  emailNotifications: true,
  notificationSender: 'PowerX IAM',
  smtpHost: '',
  smtpPort: 587,
  supportEmail: 'support@example.com',
  maintenanceMode: false,
  forceHttps: true,
  twoFactorAuth: false,
  passwordMinLength: 10,
  sessionTimeout: 30,
  maxLoginAttempts: 5,
  auditRetentionDays: 30,
  storageProvider: 'minio',
  storageBucket: 'powerx-plugin',
  retentionDays: 30,
  integrationWebhook: '',
  ssoProvider: 'powerx',
  featureFlags: {
    enableAuditStreaming: true,
    enablePortal: true,
  },
})

function FieldLabel({ children }: { children: ReactNode }) {
  return <div className="px-admin-card-text" style={{ marginBottom: 6 }}>{children}</div>
}

export default function IamSettingsPage() {
  const [activeCategory, setActiveCategory] = useState<CategoryKey>('basic')
  const [settings, setSettings] = useState<SettingsState>(defaultSettings)
  const [notice, setNotice] = useState('')

  const categories = useMemo(() => ([
    { key: 'basic' as CategoryKey, title: '基础设置', description: '站点基本信息与品牌配置', icon: '✦' },
    { key: 'security' as CategoryKey, title: '安全设置', description: '密码策略、MFA 与登录控制', icon: '🛡' },
    { key: 'notifications' as CategoryKey, title: '通知设置', description: '邮件服务、通知模板等配置', icon: '🔔' },
    { key: 'storage' as CategoryKey, title: '存储设置', description: '对象存储、备份保留策略', icon: '🗃' },
    { key: 'integrations' as CategoryKey, title: '集成设置', description: 'Webhook、SSO 与注册策略', icon: '🔗' },
    { key: 'advanced' as CategoryKey, title: '高级设置', description: '维护模式、实验特性等', icon: '🛠' },
  ]), [])

  const saveSettings = () => {
    setNotice('设置已保存（当前仅本地保存，用于对齐 UI）')
    setTimeout(() => setNotice(''), 1800)
  }

  const resetSettings = () => {
    setSettings(defaultSettings())
    setNotice('已恢复默认设置')
    setTimeout(() => setNotice(''), 1500)
  }

  return (
    <main className="px-admin-page" data-testid="iam-settings-page">
      <section className="px-admin-shell">
        <article className="px-admin-card">
          <section className="px-cap-hero">
            <p className="px-cap-kicker">ORGANIZATION & ACCESS</p>
            <h1 className="px-cap-page-title">租户配置</h1>
            <p className="px-cap-page-desc">配置站点信息、功能开关与安全策略，保持与宿主 PowerX 控制台一致的体验。</p>
          </section>

          {notice ? (
            <p className="px-alert" style={{ marginTop: 10, borderColor: '#bbf7d0', background: '#f0fdf4', color: '#166534' }}>
              {notice}
            </p>
          ) : null}

          <div style={{ marginTop: 14, display: 'grid', gap: 16, gridTemplateColumns: '300px minmax(0,1fr)' }}>
            <section className="px-admin-card" style={{ border: '1px solid #e5e7eb' }}>
              <h3 className="px-admin-card-title">配置分类</h3>
              <div style={{ marginTop: 10, display: 'grid', gap: 8 }}>
                {categories.map((category) => {
                  const active = activeCategory === category.key
                  return (
                    <button
                      key={category.key}
                      type="button"
                      onClick={() => setActiveCategory(category.key)}
                      style={{
                        width: '100%',
                        textAlign: 'left',
                        borderRadius: 12,
                        border: active ? '1px solid #86efac' : '1px solid #e5e7eb',
                        background: active ? '#f0fdf4' : '#fff',
                        padding: '12px 10px',
                        cursor: 'pointer',
                      }}
                    >
                      <div style={{ display: 'flex', gap: 10, alignItems: 'flex-start' }}>
                        <div style={{ width: 28, height: 28, borderRadius: 8, display: 'grid', placeItems: 'center', background: '#fff', border: '1px solid #dcfce7' }}>{category.icon}</div>
                        <div>
                          <div style={{ fontSize: 14, fontWeight: 700, color: '#111827' }}>{category.title}</div>
                          <div className="px-admin-card-text">{category.description}</div>
                        </div>
                      </div>
                    </button>
                  )
                })}
              </div>
            </section>

            <section className="px-admin-card" style={{ border: '1px solid #e5e7eb' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                <div>
                  <p className="px-cap-kicker" style={{ marginBottom: 2 }}>
                    {activeCategory === 'basic' ? 'GENERAL' : activeCategory.toUpperCase()}
                  </p>
                  <h3 className="px-admin-card-title">
                    {categories.find((c) => c.key === activeCategory)?.title}
                  </h3>
                </div>
                <div style={{ display: 'flex', gap: 8 }}>
                  <button type="button" className="px-btn-ghost" onClick={resetSettings}>重置</button>
                  <button type="button" className="px-btn" onClick={saveSettings}>保存</button>
                </div>
              </div>

              <div style={{ marginTop: 12, borderTop: '1px solid #e5e7eb', paddingTop: 14, display: 'grid', gap: 12 }}>
                {activeCategory === 'basic' ? (
                  <>
                    <label><FieldLabel>站点名称</FieldLabel><input className="px-field" value={settings.siteName} onChange={(e) => setSettings((s) => ({ ...s, siteName: e.target.value }))} /></label>
                    <label><FieldLabel>站点 URL</FieldLabel><input className="px-field" value={settings.siteUrl} onChange={(e) => setSettings((s) => ({ ...s, siteUrl: e.target.value }))} /></label>
                    <label><FieldLabel>管理员邮箱</FieldLabel><input className="px-field" type="email" value={settings.adminEmail} onChange={(e) => setSettings((s) => ({ ...s, adminEmail: e.target.value }))} /></label>
                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
                      <label><FieldLabel>默认语言</FieldLabel><select className="px-select" value={settings.defaultLanguage} onChange={(e) => setSettings((s) => ({ ...s, defaultLanguage: e.target.value }))}><option value="zh-CN">简体中文</option><option value="en">English</option></select></label>
                      <label><FieldLabel>时区</FieldLabel><select className="px-select" value={settings.timezone} onChange={(e) => setSettings((s) => ({ ...s, timezone: e.target.value }))}><option value="Asia/Shanghai">北京时间 (UTC+8)</option><option value="UTC">UTC</option><option value="America/Los_Angeles">美西 (UTC-8)</option></select></label>
                    </div>
                    <label><FieldLabel>日期格式</FieldLabel><select className="px-select" value={settings.dateFormat} onChange={(e) => setSettings((s) => ({ ...s, dateFormat: e.target.value }))}><option value="YYYY-MM-DD">YYYY-MM-DD</option><option value="DD/MM/YYYY">DD/MM/YYYY</option><option value="MM/DD/YYYY">MM/DD/YYYY</option></select></label>
                    <label><FieldLabel>站点描述</FieldLabel><textarea style={{ width: '100%', minHeight: 84, borderRadius: 10, border: '1px solid #cbd5e1', padding: 10, fontSize: 14 }} value={settings.siteDescription} onChange={(e) => setSettings((s) => ({ ...s, siteDescription: e.target.value }))} /></label>
                  </>
                ) : null}

                {activeCategory === 'security' ? (
                  <>
                    <label style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}><FieldLabel>强制 HTTPS</FieldLabel><input type="checkbox" checked={settings.forceHttps} onChange={(e) => setSettings((s) => ({ ...s, forceHttps: e.target.checked }))} /></label>
                    <label style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}><FieldLabel>两步验证 (MFA)</FieldLabel><input type="checkbox" checked={settings.twoFactorAuth} onChange={(e) => setSettings((s) => ({ ...s, twoFactorAuth: e.target.checked }))} /></label>
                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
                      <label><FieldLabel>密码最小长度</FieldLabel><input className="px-field" type="number" min={6} value={settings.passwordMinLength} onChange={(e) => setSettings((s) => ({ ...s, passwordMinLength: Number(e.target.value) || 10 }))} /></label>
                      <label><FieldLabel>会话超时（分钟）</FieldLabel><input className="px-field" type="number" min={5} value={settings.sessionTimeout} onChange={(e) => setSettings((s) => ({ ...s, sessionTimeout: Number(e.target.value) || 30 }))} /></label>
                    </div>
                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
                      <label><FieldLabel>登录失败次数</FieldLabel><input className="px-field" type="number" min={1} value={settings.maxLoginAttempts} onChange={(e) => setSettings((s) => ({ ...s, maxLoginAttempts: Number(e.target.value) || 5 }))} /></label>
                      <label><FieldLabel>审计日志保留天数</FieldLabel><input className="px-field" type="number" min={1} value={settings.auditRetentionDays} onChange={(e) => setSettings((s) => ({ ...s, auditRetentionDays: Number(e.target.value) || 30 }))} /></label>
                    </div>
                  </>
                ) : null}

                {activeCategory === 'notifications' ? (
                  <>
                    <label style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}><FieldLabel>启用邮件通知</FieldLabel><input type="checkbox" checked={settings.emailNotifications} onChange={(e) => setSettings((s) => ({ ...s, emailNotifications: e.target.checked }))} /></label>
                    <label><FieldLabel>发件人名称</FieldLabel><input className="px-field" value={settings.notificationSender} onChange={(e) => setSettings((s) => ({ ...s, notificationSender: e.target.value }))} /></label>
                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
                      <label><FieldLabel>SMTP Host</FieldLabel><input className="px-field" value={settings.smtpHost} onChange={(e) => setSettings((s) => ({ ...s, smtpHost: e.target.value }))} /></label>
                      <label><FieldLabel>SMTP 端口</FieldLabel><input className="px-field" type="number" value={settings.smtpPort} onChange={(e) => setSettings((s) => ({ ...s, smtpPort: Number(e.target.value) || 587 }))} /></label>
                    </div>
                    <label><FieldLabel>支持邮箱</FieldLabel><input className="px-field" type="email" value={settings.supportEmail} onChange={(e) => setSettings((s) => ({ ...s, supportEmail: e.target.value }))} /></label>
                  </>
                ) : null}

                {activeCategory === 'storage' ? (
                  <>
                    <label><FieldLabel>对象存储提供方</FieldLabel><select className="px-select" value={settings.storageProvider} onChange={(e) => setSettings((s) => ({ ...s, storageProvider: e.target.value }))}><option value="minio">MinIO</option><option value="s3">AWS S3</option><option value="local">本地存储</option></select></label>
                    <label><FieldLabel>Bucket/容器</FieldLabel><input className="px-field" value={settings.storageBucket} onChange={(e) => setSettings((s) => ({ ...s, storageBucket: e.target.value }))} /></label>
                    <label><FieldLabel>备份保留天数</FieldLabel><input className="px-field" type="number" min={1} value={settings.retentionDays} onChange={(e) => setSettings((s) => ({ ...s, retentionDays: Number(e.target.value) || 30 }))} /></label>
                  </>
                ) : null}

                {activeCategory === 'integrations' ? (
                  <>
                    <label><FieldLabel>Webhook Endpoint</FieldLabel><input className="px-field" placeholder="https://..." value={settings.integrationWebhook} onChange={(e) => setSettings((s) => ({ ...s, integrationWebhook: e.target.value }))} /></label>
                    <label><FieldLabel>SSO Provider</FieldLabel><select className="px-select" value={settings.ssoProvider} onChange={(e) => setSettings((s) => ({ ...s, ssoProvider: e.target.value }))}><option value="powerx">PowerX</option><option value="azure-ad">Azure AD</option><option value="saml">Custom SAML</option></select></label>
                    <label style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}><FieldLabel>允许自注册</FieldLabel><input type="checkbox" checked={settings.allowRegistration} onChange={(e) => setSettings((s) => ({ ...s, allowRegistration: e.target.checked }))} /></label>
                  </>
                ) : null}

                {activeCategory === 'advanced' ? (
                  <>
                    <label style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}><FieldLabel>维护模式</FieldLabel><input type="checkbox" checked={settings.maintenanceMode} onChange={(e) => setSettings((s) => ({ ...s, maintenanceMode: e.target.checked }))} /></label>
                    <label style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}><FieldLabel>启用审计流式同步</FieldLabel><input type="checkbox" checked={settings.featureFlags.enableAuditStreaming} onChange={(e) => setSettings((s) => ({ ...s, featureFlags: { ...s.featureFlags, enableAuditStreaming: e.target.checked } }))} /></label>
                    <label style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}><FieldLabel>启用 Portal 模式</FieldLabel><input type="checkbox" checked={settings.featureFlags.enablePortal} onChange={(e) => setSettings((s) => ({ ...s, featureFlags: { ...s.featureFlags, enablePortal: e.target.checked } }))} /></label>
                  </>
                ) : null}
              </div>
            </section>
          </div>
        </article>
      </section>
    </main>
  )
}
