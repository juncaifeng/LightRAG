import { useState } from 'react'
import { useDashboardStore } from '@/stores/dashboard'
import { useI18n } from '@/lib/i18n'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { ChevronDown, ChevronRight } from 'lucide-react'
import type { TopicSchema, FieldSchema } from '@/types/api'

const strategyColors: Record<string, string> = {
  APPEND: 'bg-blue-500/15 text-blue-700 dark:text-blue-400',
  REPLACE: 'bg-purple-500/15 text-purple-700 dark:text-purple-400',
  IGNORE: 'bg-gray-500/15 text-gray-700 dark:text-gray-400',
}

const pipelineColors: Record<string, string> = {
  insert: 'bg-green-500/15 text-green-700 dark:text-green-400',
  query: 'bg-orange-500/15 text-orange-700 dark:text-orange-400',
}

function desc(f: FieldSchema, lang: 'zh' | 'en') {
  return lang === 'zh' ? f.description : (f.description_en || f.description)
}

function topicDesc(schema: TopicSchema, lang: 'zh' | 'en') {
  return lang === 'zh' ? schema.description : (schema.description_en || schema.description)
}

function SchemaTable({ fields, t, lang }: { fields: FieldSchema[]; t: ReturnType<typeof useI18n>['t']; lang: 'zh' | 'en' }) {
  if (fields.length === 0) return <p className="text-sm text-muted-foreground">{t('无', 'None')}</p>

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead className="w-[160px]">{t('字段', 'Field')}</TableHead>
          <TableHead className="w-[100px]">{t('类型', 'Type')}</TableHead>
          <TableHead className="w-[60px]">{t('必填', 'Required')}</TableHead>
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
            <TableCell>
              {f.required ? (
                <Badge className="bg-red-500/15 text-red-700 dark:text-red-400 text-xs">
                  {t('是', 'yes')}
                </Badge>
              ) : (
                <span className="text-xs text-muted-foreground">{t('否', 'no')}</span>
              )}
            </TableCell>
            <TableCell className="text-sm">{desc(f, lang)}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}

function TopicCard({ schema }: { schema: TopicSchema }) {
  const { lang, t } = useI18n()
  const [expanded, setExpanded] = useState(false)

  return (
    <Card>
      <CardHeader
        className="cursor-pointer select-none py-4"
        onClick={() => setExpanded(!expanded)}
      >
        <div className="flex items-center gap-3">
          {expanded ? (
            <ChevronDown className="h-4 w-4 text-muted-foreground" />
          ) : (
            <ChevronRight className="h-4 w-4 text-muted-foreground" />
          )}
          <div className="flex-1">
            <div className="flex items-center gap-2">
              <CardTitle className="text-base font-mono">{schema.name}</CardTitle>
              <Badge className={pipelineColors[schema.pipeline] || ''} variant="secondary">
                {schema.pipeline}
              </Badge>
              <Badge variant="outline">{schema.stage}</Badge>
            </div>
            <p className="text-sm text-muted-foreground mt-1">{topicDesc(schema, lang)}</p>
          </div>
          <div className="flex items-center gap-2">
            <Badge className={strategyColors[schema.recommended_strategy] || ''}>
              {schema.recommended_strategy}
            </Badge>
            <span className="text-xs text-muted-foreground">w={schema.recommended_weight}</span>
          </div>
        </div>
      </CardHeader>

      {expanded && (
        <CardContent className="space-y-6">
          {/* Inputs */}
          <div>
            <h4 className="text-sm font-semibold mb-2">
              Inputs{' '}
              <span className="text-muted-foreground font-normal">
                {t('(publish 时传入)', '(passed during publish)')}
              </span>
            </h4>
            <SchemaTable fields={schema.inputs} t={t} lang={lang} />
          </div>

          {/* Outputs */}
          <div>
            <h4 className="text-sm font-semibold mb-2">
              Outputs{' '}
              <span className="text-muted-foreground font-normal">
                {t('(respond 时返回)', '(returned during respond)')}
              </span>
            </h4>
            <SchemaTable fields={schema.outputs} t={t} lang={lang} />
          </div>

          {/* Recommended config */}
          <div className="flex items-center gap-4 text-sm">
            <span className="text-muted-foreground">{t('推荐配置:', 'Recommended:')}</span>
            <Badge className={strategyColors[schema.recommended_strategy] || ''}>
              strategy={schema.recommended_strategy}
            </Badge>
            <Badge variant="outline">weight={schema.recommended_weight}</Badge>
          </div>
        </CardContent>
      )}
    </Card>
  )
}

export function TopicsPage() {
  const topicSchemas = useDashboardStore((s) => s.topicSchemas)
  const { t } = useI18n()

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold">{t('Topic 协议定义', 'Topic Schemas')}</h2>
        <p className="text-sm text-muted-foreground mt-1">
          {t(
            '每个 topic 的协议定义和字段说明。点击展开查看详情。',
            'Protocol definitions and field specs for each topic. Click to expand.'
          )}
        </p>
      </div>

      <div className="space-y-3">
        {topicSchemas.map((schema) => (
          <TopicCard key={schema.name} schema={schema} />
        ))}
        {topicSchemas.length === 0 && (
          <Card>
            <CardContent className="py-8 text-center text-muted-foreground">
              {t('暂无 topic 注册', 'No topic schemas registered')}
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  )
}
