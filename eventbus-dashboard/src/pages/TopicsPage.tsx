import { useState, useMemo } from 'react'
import { toast } from 'sonner'
import { useDashboardStore } from '@/stores/dashboard'
import { useI18n } from '@/lib/i18n'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { ChevronDown, ChevronRight, Copy } from 'lucide-react'
import type { TopicSchema, FieldSchema } from '@/types/api'

const strategyColors: Record<string, string> = {
  FIRST: 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-400',
  APPEND: 'bg-blue-500/15 text-blue-700 dark:text-blue-400',
  REPLACE: 'bg-purple-500/15 text-purple-700 dark:text-purple-400',
  IGNORE: 'bg-gray-500/15 text-gray-700 dark:text-gray-400',
}

const pipelineColors: Record<string, string> = {
  insert: 'bg-green-500/15 text-green-700 dark:text-green-400',
  query: 'bg-orange-500/15 text-orange-700 dark:text-orange-400',
  builder: 'bg-cyan-500/15 text-cyan-700 dark:text-cyan-400',
  retriever: 'bg-indigo-500/15 text-indigo-700 dark:text-indigo-400',
  completion: 'bg-rose-500/15 text-rose-700 dark:text-rose-400',
  merge: 'bg-amber-500/15 text-amber-700 dark:text-amber-400',
}

const domainColors: Record<string, string> = {
  rag: 'bg-sky-500/15 text-sky-700 dark:text-sky-400 border-sky-500/30',
  index: 'bg-teal-500/15 text-teal-700 dark:text-teal-400 border-teal-500/30',
  llm: 'bg-rose-500/15 text-rose-700 dark:text-rose-400 border-rose-500/30',
  kg: 'bg-amber-500/15 text-amber-700 dark:text-amber-400 border-amber-500/30',
}

function desc(f: FieldSchema, lang: 'zh' | 'en') {
  return lang === 'zh' ? f.description : (f.description_en || f.description)
}

function topicDesc(schema: TopicSchema, lang: 'zh' | 'en') {
  return lang === 'zh' ? schema.description : (schema.description_en || schema.description)
}

function copyText(text: string, label: string, t: ReturnType<typeof useI18n>['t']) {
  navigator.clipboard.writeText(text)
  toast.success(t('已复制', 'Copied') + ': ' + label)
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
              <CardTitle
                className="text-sm font-mono cursor-pointer hover:text-primary transition-colors flex items-center gap-1.5"
                title={t('点击复制主题名', 'Click to copy topic name')}
                onClick={(e) => {
                  e.stopPropagation()
                  copyText(schema.name, schema.name, t)
                }}
              >
                <span>{schema.stage}</span>
                <Copy className="h-3 w-3 text-muted-foreground opacity-0 group-hover:opacity-100" />
              </CardTitle>
              <Badge className={strategyColors[schema.recommended_strategy] || ''}>
                {schema.recommended_strategy}
              </Badge>
              <span className="text-xs text-muted-foreground">w={schema.recommended_weight}</span>
            </div>
            <p className="text-xs text-muted-foreground mt-0.5">{topicDesc(schema, lang)}</p>
          </div>
        </div>
      </CardHeader>

      {expanded && (
        <CardContent className="space-y-4 pt-0">
          {/* Full topic name */}
          <div
            className="flex items-center gap-2 px-3 py-1.5 bg-muted/50 rounded-md cursor-pointer hover:bg-muted transition-colors"
            onClick={(e) => {
              e.stopPropagation()
              copyText(schema.name, schema.name, t)
            }}
          >
            <span className="text-xs text-muted-foreground">{t('完整主题名:', 'Full topic:')}</span>
            <span className="text-xs font-mono">{schema.name}</span>
            <Copy className="h-3 w-3 text-muted-foreground ml-auto" />
          </div>

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
        </CardContent>
      )}
    </Card>
  )
}

function DomainGroup({ domain, topics }: { domain: string; topics: TopicSchema[] }) {
  const { t } = useI18n()
  const [expanded, setExpanded] = useState(true)

  const pipelineGroups = useMemo(() => {
    const map = new Map<string, TopicSchema[]>()
    for (const topic of topics) {
      const list = map.get(topic.pipeline) || []
      list.push(topic)
      map.set(topic.pipeline, list)
    }
    return Array.from(map.entries())
  }, [topics])

  return (
    <div className="space-y-2">
      <div
        className="flex items-center gap-2 cursor-pointer select-none py-1"
        onClick={() => setExpanded(!expanded)}
      >
        {expanded ? (
          <ChevronDown className="h-5 w-5 text-muted-foreground" />
        ) : (
          <ChevronRight className="h-5 w-5 text-muted-foreground" />
        )}
        <Badge className={`text-sm font-semibold px-3 py-1 ${domainColors[domain] || 'bg-gray-500/15 text-gray-700'}`} variant="outline">
          {domain}
        </Badge>
        <span className="text-sm text-muted-foreground">
          {t('{n} 个主题', '{n} topics', { n: String(topics.length) })}
        </span>
      </div>

      {expanded && (
        <div className="ml-6 space-y-4">
          {pipelineGroups.map(([pipeline, pipelineTopics]) => (
            <div key={pipeline} className="space-y-2">
              <div className="flex items-center gap-2">
                <Badge className={`text-xs font-medium ${pipelineColors[pipeline] || ''}`} variant="secondary">
                  {pipeline}
                </Badge>
                <span className="text-xs text-muted-foreground">
                  {domain}.{pipeline}.*
                </span>
              </div>
              <div className="space-y-2 ml-4">
                {pipelineTopics.map((schema) => (
                  <TopicCard key={schema.name} schema={schema} />
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

export function TopicsPage() {
  const topicSchemas = useDashboardStore((s) => s.topicSchemas)
  const { t } = useI18n()

  const domainGroups = useMemo(() => {
    const map = new Map<string, TopicSchema[]>()
    for (const schema of topicSchemas) {
      const domain = schema.name.split('.')[0]
      const list = map.get(domain) || []
      list.push(schema)
      map.set(domain, list)
    }
    return Array.from(map.entries())
  }, [topicSchemas])

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold">{t('Topic 协议定义', 'Topic Schemas')}</h2>
        <p className="text-sm text-muted-foreground mt-1">
          {t(
            '按 domain 分组展示 topic 协议定义。点击 topic 名可复制完整名称。',
            'Topics grouped by domain. Click topic name to copy full name.'
          )}
        </p>
      </div>

      <div className="space-y-6">
        {domainGroups.map(([domain, topics]) => (
          <DomainGroup key={domain} domain={domain} topics={topics} />
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
