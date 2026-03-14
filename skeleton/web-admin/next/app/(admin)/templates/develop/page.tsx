export default function TemplatesDevelopPage() {
  return (
    <main style={{ padding: 24, display: 'grid', gap: 12 }}>
      <h1 data-testid="templates-develop-title" style={{ margin: 0 }}>模板开发指引</h1>
      <p style={{ margin: 0, color: '#475569' }}>以下路径与 Nuxt 基线保持一致，用于双端联调定位。</p>
      <ul data-testid="templates-develop-structure" style={{ margin: 0, paddingLeft: 20 }}>
        <li>backend/internal/entity/models/template/template.go</li>
        <li>backend/internal/services/admin/templates/template_service.go</li>
        <li>backend/internal/transport/http/admin/templates/*</li>
        <li>web-admin/next/app/(admin)/templates/*</li>
      </ul>
    </main>
  )
}
