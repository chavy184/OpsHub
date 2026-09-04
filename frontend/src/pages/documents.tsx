import { useEffect, useState, useCallback, useRef } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select } from '@/components/ui/select'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { FolderPlus, Upload, Trash2, Download, FileText, Folder, FolderOpen, ChevronRight, ChevronDown, Plus, X } from 'lucide-react'
import { toast } from 'sonner'
import {
  getDocumentTree,
  uploadDocument,
  createFolder,
  deleteDocument,
  getDocumentContent,
  getDownloadUrl,
  listDocTypes,
  createDocType,
  deleteDocType,
  type FileNode,
  type FileContent,
  type DocCategory,
} from '@/api/documents'

// 可预览的扩展名
const PREVIEWABLE_EXTS = new Set([
  '.md', '.txt', '.yaml', '.yml', '.json', '.xml', '.html', '.css',
  '.js', '.ts', '.tsx', '.jsx', '.go', '.py', '.sh', '.sql',
  '.toml', '.ini', '.conf', '.cfg', '.env', '.log', '.csv',
])

function getFileExt(name: string): string {
  const idx = name.lastIndexOf('.')
  return idx >= 0 ? name.slice(idx).toLowerCase() : ''
}

function isPreviewable(name: string): boolean {
  return PREVIEWABLE_EXTS.has(getFileExt(name))
}

function formatSize(bytes: number): string {
  if (bytes === 0) return '-'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

// --- 文件树节点组件 ---
function TreeNode({
  node,
  depth,
  selectedPath,
  onSelect,
}: {
  node: FileNode
  depth: number
  selectedPath: string | null
  onSelect: (node: FileNode) => void
}) {
  const [expanded, setExpanded] = useState(depth < 1)
  const isSelected = selectedPath === node.path

  if (node.is_dir) {
    return (
      <div>
        <div
          className={`flex items-center gap-1 px-2 py-1 cursor-pointer rounded text-sm hover:bg-accent ${isSelected ? 'bg-accent' : ''}`}
          style={{ paddingLeft: `${depth * 16 + 8}px` }}
          onClick={() => {
            setExpanded(!expanded)
            onSelect(node)
          }}
        >
          {expanded ? <ChevronDown className="w-3.5 h-3.5 shrink-0" /> : <ChevronRight className="w-3.5 h-3.5 shrink-0" />}
          {expanded ? <FolderOpen className="w-4 h-4 shrink-0 text-amber-500" /> : <Folder className="w-4 h-4 shrink-0 text-amber-500" />}
          <span className="truncate">{node.name}</span>
        </div>
        {expanded && node.children?.map((child) => (
          <TreeNode key={child.path} node={child} depth={depth + 1} selectedPath={selectedPath} onSelect={onSelect} />
        ))}
      </div>
    )
  }

  return (
    <div
      className={`flex items-center gap-1 px-2 py-1 cursor-pointer rounded text-sm hover:bg-accent ${isSelected ? 'bg-accent font-medium' : ''}`}
      style={{ paddingLeft: `${depth * 16 + 8}px` }}
      onClick={() => onSelect(node)}
    >
      <FileText className="w-4 h-4 shrink-0 text-muted-foreground" />
      <span className="truncate">{node.name}</span>
    </div>
  )
}

// --- 主页面 ---
export default function DocumentsPage() {
  const [docTypes, setDocTypes] = useState<DocCategory[]>([])
  const [docType, setDocType] = useState<string>('')
  const [tree, setTree] = useState<FileNode[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedNode, setSelectedNode] = useState<FileNode | null>(null)
  const [preview, setPreview] = useState<FileContent | null>(null)
  const [previewLoading, setPreviewLoading] = useState(false)

  // 对话框
  const [folderDialogOpen, setFolderDialogOpen] = useState(false)
  const [folderName, setFolderName] = useState('')
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [overwriteDialogOpen, setOverwriteDialogOpen] = useState(false)
  const [pendingFile, setPendingFile] = useState<File | null>(null)
  const [typeDialogOpen, setTypeDialogOpen] = useState(false)
  const [newTypeName, setNewTypeName] = useState('')
  const [deleteTypeDialogOpen, setDeleteTypeDialogOpen] = useState(false)

  const fileInputRef = useRef<HTMLInputElement>(null)

  // 加载文档类型列表
  const fetchTypes = useCallback(async () => {
    try {
      const types = await listDocTypes()
      setDocTypes(types)
      if (types.length > 0 && !docType) {
        setDocType(types[0]!.name)
      }
    } catch {
      toast.error('获取文档类型失败')
    }
  }, [docType])

  useEffect(() => { fetchTypes() }, [])

  const fetchTree = useCallback(async () => {
    if (!docType) { setTree([]); setLoading(false); return }
    setLoading(true)
    try {
      const data = await getDocumentTree(docType)
      setTree(data)
    } catch {
      toast.error('获取文档列表失败')
    } finally {
      setLoading(false)
    }
  }, [docType])

  useEffect(() => {
    fetchTree()
    setSelectedNode(null)
    setPreview(null)
  }, [fetchTree])

  // 选中文件时预览
  const handleSelect = useCallback(async (node: FileNode) => {
    setSelectedNode(node)
    if (!node.is_dir && isPreviewable(node.name)) {
      setPreviewLoading(true)
      try {
        const content = await getDocumentContent(docType, node.path)
        setPreview(content)
      } catch {
        setPreview(null)
      } finally {
        setPreviewLoading(false)
      }
    } else {
      setPreview(null)
    }
  }, [docType])

  // 上传
  const handleUpload = useCallback(async (file: File, overwrite = false) => {
    const targetDir = selectedNode?.is_dir ? selectedNode.path : ''
    try {
      await uploadDocument(docType, targetDir, file, overwrite)
      toast.success(`上传成功: ${file.name}`)
      fetchTree()
    } catch (err: any) {
      if (err?.status === 409 && !overwrite) {
        setPendingFile(file)
        setOverwriteDialogOpen(true)
      } else {
        toast.error(err?.message || '上传失败')
      }
    }
  }, [docType, selectedNode, fetchTree])

  const handleFileChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files
    if (!files || files.length === 0) return
    Array.from(files).forEach((file) => handleUpload(file))
    e.target.value = ''
  }, [handleUpload])

  // 新建文件夹
  const handleCreateFolder = useCallback(async () => {
    if (!folderName.trim()) return
    const parentDir = selectedNode?.is_dir ? selectedNode.path : ''
    const path = parentDir ? `${parentDir}/${folderName.trim()}` : folderName.trim()
    try {
      await createFolder(docType, path)
      toast.success(`文件夹已创建: ${folderName}`)
      setFolderDialogOpen(false)
      setFolderName('')
      fetchTree()
    } catch (err: any) {
      toast.error(err?.message || '创建失败')
    }
  }, [docType, selectedNode, folderName, fetchTree])

  // 删除
  const handleDelete = useCallback(async () => {
    if (!selectedNode) return
    try {
      await deleteDocument(docType, selectedNode.path)
      toast.success(`已删除: ${selectedNode.name}`)
      setDeleteDialogOpen(false)
      setSelectedNode(null)
      setPreview(null)
      fetchTree()
    } catch (err: any) {
      toast.error(err?.message || '删除失败')
    }
  }, [docType, selectedNode, fetchTree])

  // 下载
  const handleDownload = useCallback(() => {
    if (!selectedNode || selectedNode.is_dir) return
    const url = getDownloadUrl(docType, selectedNode.path)
    window.open(url, '_blank')
  }, [docType, selectedNode])

  // 创建文档类型
  const handleCreateType = useCallback(async () => {
    if (!newTypeName.trim()) return
    try {
      await createDocType(newTypeName.trim())
      toast.success(`文档类型已创建: ${newTypeName}`)
      setTypeDialogOpen(false)
      setNewTypeName('')
      const types = await listDocTypes()
      setDocTypes(types)
      setDocType(newTypeName.trim())
    } catch (err: any) {
      toast.error(err?.message || '创建失败')
    }
  }, [newTypeName])

  // 删除文档类型
  const handleDeleteType = useCallback(async () => {
    if (!docType) return
    try {
      await deleteDocType(docType)
      toast.success(`文档类型已删除: ${docType}`)
      setDeleteTypeDialogOpen(false)
      setDocType('')
      setTree([])
      setSelectedNode(null)
      setPreview(null)
      const types = await listDocTypes()
      setDocTypes(types)
      if (types.length > 0) setDocType(types[0]!.name)
    } catch (err: any) {
      toast.error(err?.message || '删除失败')
    }
  }, [docType])

  return (
    <div className="h-full flex flex-col">
      {/* 顶部操作栏 */}
      <div className="flex items-center justify-between p-4 border-b">
        <h1 className="text-xl font-semibold">文档管理</h1>
        <div className="flex items-center gap-2">
          {docTypes.length > 0 && (
            <Select
              options={docTypes.map(t => ({ value: t.name, label: t.name }))}
              value={docType}
              onChange={(e) => setDocType(e.target.value)}
            />
          )}
          <Button variant="outline" size="sm" onClick={() => setTypeDialogOpen(true)} title="新建文档类型">
            <Plus className="w-4 h-4 mr-1" /> 新建类型
          </Button>
          {docType && (
            <Button variant="ghost" size="sm" onClick={() => setDeleteTypeDialogOpen(true)} title="删除当前类型" className="text-destructive">
              <X className="w-4 h-4" />
            </Button>
          )}
        </div>
      </div>

      {/* 主内容区 */}
      <div className="flex flex-1 overflow-hidden">
        {/* 左侧：文件树 */}
        <div className="w-72 border-r flex flex-col">
          <div className="flex items-center gap-1 p-2 border-b">
            <Button variant="ghost" size="sm" onClick={() => setFolderDialogOpen(true)} title="新建文件夹">
              <FolderPlus className="w-4 h-4" />
            </Button>
            <Button variant="ghost" size="sm" onClick={() => fileInputRef.current?.click()} title="上传文件">
              <Upload className="w-4 h-4" />
            </Button>
            {selectedNode && (
              <>
                {!selectedNode.is_dir && (
                  <Button variant="ghost" size="sm" onClick={handleDownload} title="下载">
                    <Download className="w-4 h-4" />
                  </Button>
                )}
                <Button variant="ghost" size="sm" onClick={() => setDeleteDialogOpen(true)} title="删除" className="text-destructive">
                  <Trash2 className="w-4 h-4" />
                </Button>
              </>
            )}
          </div>

          <div className="flex-1 overflow-y-auto p-1">
            {loading ? (
              <div className="space-y-2 p-2">
                <Skeleton className="h-5 w-full" />
                <Skeleton className="h-5 w-3/4" />
                <Skeleton className="h-5 w-2/3" />
              </div>
            ) : tree.length === 0 ? (
              <div className="text-sm text-muted-foreground text-center py-8">
                暂无文档，点击上方按钮上传
              </div>
            ) : (
              tree.map((node) => (
                <TreeNode key={node.path} node={node} depth={0} selectedPath={selectedNode?.path ?? null} onSelect={handleSelect} />
              ))
            )}
          </div>

          <input
            ref={fileInputRef}
            type="file"
            multiple
            className="hidden"
            onChange={handleFileChange}
          />
        </div>

        {/* 右侧：预览 */}
        <div className="flex-1 overflow-y-auto p-4">
          {previewLoading ? (
            <div className="space-y-2">
              <Skeleton className="h-6 w-1/3" />
              <Skeleton className="h-4 w-full" />
              <Skeleton className="h-4 w-2/3" />
            </div>
          ) : preview ? (
            <div>
              <div className="flex items-center justify-between mb-4">
                <h2 className="text-lg font-medium">{selectedNode?.name}</h2>
                <span className="text-sm text-muted-foreground">{formatSize(preview.size)}</span>
              </div>
              <pre className="text-sm bg-muted p-4 rounded-lg overflow-x-auto whitespace-pre-wrap break-words font-mono">
                {preview.content}
              </pre>
            </div>
          ) : selectedNode ? (
            <div className="text-center text-muted-foreground py-16">
              {selectedNode.is_dir ? (
                <div>
                  <Folder className="w-12 h-12 mx-auto mb-2 text-amber-500" />
                  <p>文件夹: {selectedNode.name}</p>
                  <p className="text-xs mt-1">{selectedNode.children?.length ?? 0} 个项目</p>
                </div>
              ) : (
                <div>
                  <FileText className="w-12 h-12 mx-auto mb-2" />
                  <p>{selectedNode.name}</p>
                  <p className="text-xs mt-1">{formatSize(selectedNode.size)} · 不支持预览此文件类型</p>
                  <Button variant="outline" size="sm" className="mt-4" onClick={handleDownload}>
                    <Download className="w-4 h-4 mr-1" /> 下载文件
                  </Button>
                </div>
              )}
            </div>
          ) : (
            <div className="text-center text-muted-foreground py-16">
              <FileText className="w-12 h-12 mx-auto mb-2" />
              <p>选择文件查看内容</p>
            </div>
          )}
        </div>
      </div>

      {/* 新建文件夹对话框 */}
      <Dialog open={folderDialogOpen} onOpenChange={setFolderDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>新建文件夹</DialogTitle>
          </DialogHeader>
          <Input
            placeholder="文件夹名称"
            value={folderName}
            onChange={(e) => setFolderName(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleCreateFolder()}
          />
          <DialogFooter>
            <Button variant="outline" onClick={() => setFolderDialogOpen(false)}>取消</Button>
            <Button onClick={handleCreateFolder}>创建</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 删除确认对话框 */}
      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确认删除</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">
            确定要删除 <span className="font-medium text-foreground">{selectedNode?.name}</span> 吗？
            {selectedNode?.is_dir && '文件夹内的所有内容将被一并删除。'}
            此操作不可撤销。
          </p>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteDialogOpen(false)}>取消</Button>
            <Button variant="destructive" onClick={handleDelete}>删除</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 覆盖确认对话框 */}
      <Dialog open={overwriteDialogOpen} onOpenChange={setOverwriteDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>文件已存在</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">
            目标位置已存在同名文件 <span className="font-medium text-foreground">{pendingFile?.name}</span>，是否覆盖？
          </p>
          <DialogFooter>
            <Button variant="outline" onClick={() => { setOverwriteDialogOpen(false); setPendingFile(null) }}>跳过</Button>
            <Button onClick={() => {
              setOverwriteDialogOpen(false)
              if (pendingFile) {
                handleUpload(pendingFile, true)
                setPendingFile(null)
              }
            }}>覆盖</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 新建文档类型对话框 */}
      <Dialog open={typeDialogOpen} onOpenChange={setTypeDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>新建文档类型</DialogTitle>
          </DialogHeader>
          <Input
            placeholder="类型名称，如：项目文档、Skills、API文档"
            value={newTypeName}
            onChange={(e) => setNewTypeName(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleCreateType()}
          />
          <DialogFooter>
            <Button variant="outline" onClick={() => setTypeDialogOpen(false)}>取消</Button>
            <Button onClick={handleCreateType}>创建</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 删除文档类型确认对话框 */}
      <Dialog open={deleteTypeDialogOpen} onOpenChange={setDeleteTypeDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>删除文档类型</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">
            确定要删除文档类型 <span className="font-medium text-foreground">{docType}</span> 吗？
            该类型下的所有文件和文件夹将被永久删除。此操作不可撤销。
          </p>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteTypeDialogOpen(false)}>取消</Button>
            <Button variant="destructive" onClick={handleDeleteType}>删除</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
