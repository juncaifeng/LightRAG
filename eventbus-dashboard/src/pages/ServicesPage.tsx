import { useState } from 'react'
import { toast } from 'sonner'
import { useDashboardStore } from '@/stores/dashboard'
import { useI18n } from '@/lib/i18n'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { ChevronDown, ChevronRight, Copy } from 'lucide-react'
import type { ServiceSchema, MethodSchema, FieldSchema } from '@/types/api'

function desc(text: string, textEn: string, lang: 'zh' | 'en') {
  return lang === 'zh' ? text : (textEn || text)
}

function copyText(text: string, label: string, t: ReturnType<typeof useI18n>['t']) {
  navigator.clipboard.writeText(text)
  toast.success(t('已复制', 'Copied') + ': ' + label)
}

function FieldTable({ fields, t, lang }: { fields: FieldSchema[]; t: ReturnType<typeof useI18n>['t']; lang: 'zh' | 'en' }) {
  if (fields.length === 0) return <p className="text-sm text-muted-foreground">{t('无', 'None')}</p>

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead className="w-[160px]">{t('字段', 'Field')}</TableHead>
          <TableHead className="w-[100px]">{t('类型', 'Type')}</TableHead>
          <TableHead>{t('说明', 'Description')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {fields.map((f) => (
          <TableRow key={f.name}>
            <TableCell className="font-mono text-sm">{f.name}</TableCell>
            <TableCell>
              <Badge variant="outline" className="font-mono text-xs">{f.type}</Badge>
            </TableCell>
            <TableCell className="text-sm">{desc(f.description, f.description_en, lang)}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}

function MethodRow({ method, t, lang }: { method: MethodSchema; t: ReturnType<typeof useI18n>['t']; lang: 'zh' | 'en' }) {
  return (
    <div className="flex items-center gap-3 py-1.5 px-3 rounded-md hover:bg-muted/50 transition-colors">
      <Badge variant="secondary" className="font-mono text-xs shrink-0">rpc</Badge>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-sm font-mono font-medium">{method.name}</span>
          <span className="text-xs text-muted-foreground">
            ({method.input_type}) → ({method.output_type})
          </span>
        </div>
        {(method.description || method.description_en) && (
          <p className="text-xs text-muted-foreground mt-0.5">
            {desc(method.description, method.description_en, lang)}
          </p>
        )}
      </div>
    </div>
  )
}

function MessageCard({ message, t, lang }: { message: { name: string; description: string; fields: FieldSchema[] }; t: ReturnType<typeof useI18n>['t']; lang: 'zh' | 'en' }) {
  const [expanded, setExpanded] = useState(false)

  return (
    <Card className="overflow-hidden">
      <CardHeader
        className="cursor-pointer select-none py-2.5"
        onClick={() => setExpanded(!expanded)}
      >
        <div className="flex items-center gap-2">
          {expanded ? (
            <ChevronDown className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
          ) : (
            <ChevronRight className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
          )}
          <CardTitle className="text-sm font-mono">{message.name}</CardTitle>
          {message.description && (
            <span className="text-xs text-muted-foreground truncate">{message.description}</span>
          )}
        </div>
      </CardHeader>
      {expanded && (
        <CardContent className="pt-0">
          <FieldTable fields={message.fields} t={t} lang={lang} />
        </CardContent>
      )}
    </Card>
  )
}

function ServiceCard({ schema }: { schema: ServiceSchema }) {
  const { lang, t } = useI18n()
  const [expanded, setExpanded] = useState(false)

  return (
    <Card>
      <CardHeader
        className="cursor-pointer select-none py-3"
        onClick={() => setExpanded(!expanded)}
      >
        <div className="flex items-center gap-3">
          {expanded ? (
            <ChevronDown className="h-4 w-4 text-muted-foreground shrink-0" />
          ) : (
            <ChevronRight className="h-4 w-4 text-muted-foreground shrink-0" />
          )}
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 flex-wrap">
              <CardTitle className="text-sm font-mono">{schema.name}</CardTitle>
              <Badge className="bg-indigo-500/15 text-indigo-700 dark:text-indigo-400 text-xs">
                gRPC
              </Badge>
              <span className="text-xs text-muted-foreground">{schema.methods.length} {t('个方法', 'methods')}</span>
            </div>
            {(schema.description || schema.description_en) && (
              <p className="text-xs text-muted-foreground mt-0.5">
                {desc(schema.description, schema.description_en, lang)}
              </p>
            )}
          </div>
        </div>
      </CardHeader>

      {expanded && (
        <CardContent className="space-y-4 pt-0">
          {/* Package */}
          <div
            className="flex items-center gap-2 px-3 py-1.5 bg-muted/50 rounded-md cursor-pointer hover:bg-muted transition-colors"
            onClick={(e) => {
              e.stopPropagation()
              copyText(schema.package, schema.package, t)
            }}
          >
            <span className="text-xs text-muted-foreground">{t('包名:', 'Package:')}</span>
            <span className="text-xs font-mono">{schema.package}</span>
            <Copy className="h-3 w-3 text-muted-foreground ml-auto" />
          </div>

          {/* Methods */}
          <div>
            <h4 className="text-sm font-semibold mb-2">
              {t('RPC 方法', 'RPC Methods')}
            </h4>
            <div className="space-y-1">
              {schema.methods.map((method) => (
                <MethodRow key={method.name} method={method} t={t} lang={lang} />
              ))}
            </div>
          </div>

          {/* Messages */}
          <div>
            <h4 className="text-sm font-semibold mb-2">
              {t('消息类型', 'Messages')}
              <span className="text-muted-foreground font-normal ml-1">
                ({schema.messages.length})
              </span>
            </h4>
            <div className="space-y-2">
              {schema.messages.map((msg) => (
                <MessageCard key={msg.name} message={msg} t={t} lang={lang} />
              ))}
            </div>
          </div>
        </CardContent>
      )}
    </Card>
  )
}

export function ServicesPage() {
  const serviceSchemas = useDashboardStore((s) => s.serviceSchemas)
  const { t } = useI18n()

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold">{t('gRPC 服务定义', 'gRPC Service Schemas')}</h2>
        <p className="text-sm text-muted-foreground mt-1">
          {t(
            '注册的 gRPC 服务协议定义。点击服务名展开查看方法和消息类型。',
            'Registered gRPC service protocol definitions. Click to expand methods and message types.'
          )}
        </p>
      </div>

      <div className="space-y-4">
        {serviceSchemas.map((schema) => (
          <ServiceCard key={schema.name} schema={schema} />
        ))}
        {serviceSchemas.length === 0 && (
          <Card>
            <CardContent className="py-8 text-center text-muted-foreground">
              {t('暂无服务注册', 'No services registered')}
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  )
}
