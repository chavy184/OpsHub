import { useMemo, useRef, useState } from 'react'
import { Archive, ChevronDown, ChevronRight, FileText, Folder, FolderOpen, Info, Upload } from 'lucide-react'
import { toast } from 'sonner'
import { analyzeArchive, type ArchiveAnalysis, type ArchiveNode } from '@/api/archives'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'

function formatSize(bytes: number): string {
  if (!bytes) return '-'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function flattenTextFiles(nodes: ArchiveNode[]): ArchiveNode[] {
  const result: ArchiveNode[] = []
  const walk = (items: ArchiveNode[]) => {
    items.forEach((item) => {
      if (item.content) result.push(item)
      if (item.children) walk(item.children)
    })
  }
  walk(nodes)
  return result
}

function PackageField({ label, value }: { label: string; value?: string }) {
  if (!value) return null
  return (
    <div className="min-w-0">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="truncate text-sm font-medium">{value}</div>
    </div>
  )
}

function TreeNode({
  node,
  depth,
  selectedPath,
  onSelect,
}: {
  node: ArchiveNode
  depth: number
  selectedPath: string | null
  onSelect: (node: ArchiveNode) => void
}) {
  const [expanded, setExpanded] = useState(depth < 1)
  const selected = selectedPath === `${node.section}:${node.path}`
  const isDir = node.kind === 'dir'
  return (
    <div>
      <button
        type="button"
        className={`flex h-8 w-full items-center gap-1 rounded px-2 text-left text-sm hover:bg-accent ${selected ? 'bg-accent font-medium' : ''}`}
        style={{ paddingLeft: `${depth * 14 + 8}px` }}
        onClick={() => {
          if (isDir) setExpanded((v) => !v)
          onSelect(node)
        }}
      >
        {isDir ? (
          expanded ? <ChevronDown className="size-3.5 shrink-0" /> : <ChevronRight className="size-3.5 shrink-0" />
        ) : (
          <span className="w-3.5 shrink-0" />
        )}
        {isDir ? (
          expanded ? <FolderOpen className="size-4 shrink-0 text-amber-500" /> : <Folder className="size-4 shrink-0 text-amber-500" />
        ) : (
          <FileText className="size-4 shrink-0 text-muted-foreground" />
        )}
        <span className="min-w-0 flex-1 truncate">{node.name}</span>
        {node.content && <Badge variant="secondary" className="px-1.5 py-0 text-[10px]">文本</Badge>}
      </button>
      {isDir && expanded && node.children?.map((child) => (
        <TreeNode
          key={`${child.section}:${child.path}`}
          node={child}
          depth={depth + 1}
          selectedPath={selectedPath}
          onSelect={onSelect}
        />
      ))}
    </div>
  )
}

export default function ArchivesPage() {
  const inputRef = useRef<HTMLInputElement>(null)
  const [analysis, setAnalysis] = useState<ArchiveAnalysis | null>(null)
  const [selectedNode, setSelectedNode] = useState<ArchiveNode | null>(null)
  const [loading, setLoading] = useState(false)
  const selectedKey = selectedNode ? `${selectedNode.section}:${selectedNode.path}` : null
  const textFiles = useMemo(() => flattenTextFiles(analysis?.files ?? []), [analysis])
  const info = analysis?.package_info ?? {}

  const handleFile = async (file: File) => {
    setLoading(true)
    setSelectedNode(null)
    try {
      const result = await analyzeArchive(file)
      setAnalysis(result)
      const firstText = flattenTextFiles(result.files)[0]
      if (firstText) setSelectedNode(firstText)
      toast.success('解析完成')
    } catch (err: any) {
      toast.error(err?.message || '解析失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex h-[calc(100vh-7rem)] flex-col overflow-hidden">
      <div className="flex items-center justify-between border-b pb-4">
        <div>
          <h1 className="text-xl font-semibold">解压目录</h1>
          <p className="mt-1 text-sm text-muted-foreground">上传 deb 包后查看包结构和可解析文本内容</p>
        </div>
        <div className="flex items-center gap-2">
          {analysis && <Badge variant="outline">{analysis.type}</Badge>}
          <Button onClick={() => inputRef.current?.click()} disabled={loading}>
            <Upload className="size-4" />
            上传文件
          </Button>
          <input
            ref={inputRef}
            type="file"
            accept=".deb"
            className="hidden"
            onChange={(event) => {
              const file = event.target.files?.[0]
              if (file) handleFile(file)
              event.target.value = ''
            }}
          />
        </div>
      </div>

      {loading ? (
        <div className="grid flex-1 grid-cols-[320px_1fr] gap-4 overflow-hidden py-4">
          <div className="space-y-2 border-r pr-4">
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-5/6" />
            <Skeleton className="h-8 w-3/4" />
          </div>
          <div className="space-y-3">
            <Skeleton className="h-8 w-1/3" />
            <Skeleton className="h-40 w-full" />
          </div>
        </div>
      ) : analysis ? (
        <div className="grid min-h-0 flex-1 grid-cols-[320px_1fr] gap-4 overflow-hidden py-4">
          <aside className="flex min-h-0 flex-col overflow-hidden border-r pr-4">
            <div className="mb-3 grid grid-cols-2 gap-3 rounded-md border bg-muted/20 p-3">
              <PackageField label="文件" value={analysis.filename} />
              <PackageField label="大小" value={formatSize(analysis.size)} />
              <PackageField label="包名" value={info.Package} />
              <PackageField label="版本" value={info.Version} />
            </div>
            <div className="mb-2 flex items-center justify-between text-xs text-muted-foreground">
              <span>包结构</span>
              <span>{textFiles.length} 个文本</span>
            </div>
            <div className="min-h-0 flex-1 overflow-y-auto pr-1">
              {analysis.files.map((node) => (
                <TreeNode
                  key={`${node.section}:${node.path}`}
                  node={node}
                  depth={0}
                  selectedPath={selectedKey}
                  onSelect={setSelectedNode}
                />
              ))}
            </div>
          </aside>

          <main className="min-h-0 overflow-y-auto">
            {analysis.warnings && analysis.warnings.length > 0 && (
              <div className="mb-4 rounded-md border border-amber-200 bg-amber-50 p-3 text-sm text-amber-800">
                {analysis.warnings.map((warning) => <div key={warning}>{warning}</div>)}
              </div>
            )}

            {selectedNode ? (
              <div className="space-y-4">
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <h2 className="truncate text-lg font-medium">{selectedNode.path}</h2>
                    <div className="mt-1 flex items-center gap-2 text-sm text-muted-foreground">
                      <Badge variant="secondary">{selectedNode.section}</Badge>
                      <span>{selectedNode.kind === 'dir' ? '目录' : formatSize(selectedNode.size)}</span>
                      {selectedNode.mode && <span>权限 {selectedNode.mode}</span>}
                    </div>
                  </div>
                </div>

                {selectedNode.content ? (
                  <pre className="max-h-[calc(100vh-18rem)] overflow-auto rounded-md border bg-muted/30 p-4 font-mono text-sm whitespace-pre-wrap break-words">
                    {selectedNode.content}
                  </pre>
                ) : selectedNode.kind === 'dir' ? (
                  <div className="rounded-md border p-8 text-center text-sm text-muted-foreground">
                    <Folder className="mx-auto mb-2 size-10 text-amber-500" />
                    该目录包含 {selectedNode.children?.length ?? 0} 个条目
                  </div>
                ) : (
                  <div className="rounded-md border p-8 text-center text-sm text-muted-foreground">
                    <FileText className="mx-auto mb-2 size-10" />
                    当前文件没有可预览文本内容
                  </div>
                )}
              </div>
            ) : (
              <div className="rounded-md border p-10 text-center text-muted-foreground">
                <Info className="mx-auto mb-2 size-10" />
                选择左侧条目查看内容
              </div>
            )}
          </main>
        </div>
      ) : (
        <div className="flex flex-1 items-center justify-center">
          <button
            type="button"
            onClick={() => inputRef.current?.click()}
            className="flex w-full max-w-xl flex-col items-center rounded-md border border-dashed p-12 text-center transition-colors hover:bg-muted/40"
          >
            <Archive className="mb-3 size-12 text-muted-foreground" />
            <span className="text-base font-medium">上传 deb 包开始解析</span>
            <span className="mt-1 text-sm text-muted-foreground">当前支持 Debian package，后续可扩展更多归档格式</span>
          </button>
        </div>
      )}
    </div>
  )
}
