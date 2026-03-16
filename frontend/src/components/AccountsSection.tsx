import { useState, useEffect, useRef, useCallback } from 'react'
import { DragDropProvider, useDraggable, useDroppable } from '@dnd-kit/react'
import {
  getAccounts,
  updateAccount,
  createGroup,
  addGroupMember,
  removeGroupMember,
  deleteGroup as deleteGroupApi,
  updateGroup,
  type AccountItem,
  type AccountsResponse,
  type UpdateAccountRequest,
  type GroupItem,
} from '../api/client'
import { getAccountDisplayName } from '../utils/account'
import { formatCurrency } from '../utils/format'

interface AccountsSectionProps {
  onAccountRestored?: (names: string[]) => void
}

type PanelType = 'liquid' | 'savings' | 'investments' | 'other'

const PANEL_LABELS: Record<PanelType, string> = {
  liquid: 'Liquid',
  savings: 'Savings',
  investments: 'Investments',
  other: 'Other',
}

const PANEL_TYPE_TO_OVERRIDE: Record<PanelType, string> = {
  liquid: 'checking',
  savings: 'savings',
  investments: 'investment',
  other: 'other',
}

const PANEL_ORDER: PanelType[] = ['liquid', 'savings', 'investments', 'other']

// --- Icons (inline SVGs, 16x16) ---

function PencilIcon() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z" />
      <path d="m15 5 4 4" />
    </svg>
  )
}

function EyeOffIcon() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M10.73 5.08A10.43 10.43 0 0 1 12 5c7 0 10 7 10 7a13.16 13.16 0 0 1-1.67 2.68" />
      <path d="M6.61 6.61A13.53 13.53 0 0 0 2 12s3 7 10 7a9.74 9.74 0 0 0 5.39-1.61" />
      <line x1="2" x2="22" y1="2" y2="22" />
    </svg>
  )
}

function EyeIcon() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z" />
      <circle cx="12" cy="12" r="3" />
    </svg>
  )
}

function GripIcon() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="9" cy="5" r="1" /><circle cx="15" cy="5" r="1" />
      <circle cx="9" cy="12" r="1" /><circle cx="15" cy="12" r="1" />
      <circle cx="9" cy="19" r="1" /><circle cx="15" cy="19" r="1" />
    </svg>
  )
}

function ResetIcon() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8" />
      <path d="M3 3v5h5" />
    </svg>
  )
}

function ChevronIcon({ expanded }: { expanded: boolean }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24"
      fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"
      className={`transition-transform ${expanded ? 'rotate-90' : ''}`}
      aria-hidden="true"
    >
      <path d="m9 18 6-6-6-6" />
    </svg>
  )
}

// --- Draggable Account Row ---

function DraggableAccountRow({
  account,
  panelType,
  editingId,
  editValue,
  savingId,
  errorId,
  isMobile,
  onStartEdit,
  onEditChange,
  onSaveEdit,
  onCancelEdit,
  onReset,
  onHide,
  onTypeChange,
}: {
  account: AccountItem
  panelType: PanelType
  editingId: string | null
  editValue: string
  savingId: string | null
  errorId: string | null
  isMobile: boolean
  onStartEdit: (id: string, currentName: string) => void
  onEditChange: (value: string) => void
  onSaveEdit: (id: string) => void
  onCancelEdit: () => void
  onReset: (id: string) => void
  onHide: (id: string) => void
  onTypeChange: (id: string, newType: PanelType) => void
}) {
  const isEditing = editingId === account.id
  const isSaving = savingId === account.id
  const hasError = errorId === account.id
  const inputRef = useRef<HTMLInputElement>(null)

  const { ref: draggableRef, isDragging } = useDraggable({
    id: account.id,
    data: { panelType },
    disabled: isMobile,
  })

  useEffect(() => {
    if (isEditing && inputRef.current) {
      inputRef.current.focus()
    }
  }, [isEditing])

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === 'Enter') {
      e.preventDefault()
      onSaveEdit(account.id)
    } else if (e.key === 'Escape') {
      e.preventDefault()
      onCancelEdit()
    }
  }

  return (
    <div
      ref={draggableRef}
      data-testid={`account-row-${account.id}`}
      className={`flex items-center gap-2 py-2 px-3 rounded-md hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors ${
        isSaving ? 'opacity-50' : ''
      } ${isDragging ? 'opacity-30' : ''}`}
    >
      {/* Drag handle (desktop only) */}
      {!isMobile && (
        <span className="cursor-grab text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 shrink-0" title="Drag to move">
          <GripIcon />
        </span>
      )}

      {/* Account info */}
      <div className="flex-1 min-w-0">
        {isEditing ? (
          <input
            ref={inputRef}
            type="text"
            value={editValue}
            onChange={(e) => onEditChange(e.target.value)}
            onKeyDown={handleKeyDown}
            onBlur={() => onSaveEdit(account.id)}
            placeholder={account.original_name}
            className="w-full border-b-2 border-blue-500 bg-transparent text-sm text-gray-900 dark:text-gray-100 focus:outline-none py-0.5"
            aria-label="Edit account name"
          />
        ) : (
          <div>
            <span className="text-sm text-gray-900 dark:text-gray-100 truncate block">
              {getAccountDisplayName(account)}
            </span>
            {account.display_name && (
              <span className="text-xs text-gray-400 dark:text-gray-500 truncate block">
                {account.original_name}
              </span>
            )}
          </div>
        )}
        {hasError && (
          <span className="text-xs text-red-500">Failed to save</span>
        )}
      </div>

      {/* Balance */}
      <span className="text-sm text-gray-700 dark:text-gray-300 font-medium shrink-0">
        {formatCurrency(account.balance)}
      </span>

      {/* Action buttons */}
      <div className="flex items-center gap-1 shrink-0">
        {!isEditing && (
          <button
            type="button"
            onClick={() => onStartEdit(account.id, account.display_name ?? '')}
            className="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
            title="Edit name"
            aria-label="Edit name"
          >
            <PencilIcon />
          </button>
        )}

        {account.display_name && !isEditing && (
          <button
            type="button"
            onClick={() => onReset(account.id)}
            className="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
            title="Reset to original name"
            aria-label="Reset name"
          >
            <ResetIcon />
          </button>
        )}

        <button
          type="button"
          onClick={() => onHide(account.id)}
          className="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
          title="Hide account"
          aria-label="Hide account"
        >
          <EyeOffIcon />
        </button>

        {/* Mobile type dropdown */}
        {isMobile && (
          <select
            value={panelType}
            onChange={(e) => onTypeChange(account.id, e.target.value as PanelType)}
            className="text-xs border border-gray-300 dark:border-gray-600 rounded px-1 py-0.5 bg-white dark:bg-gray-800 text-gray-700 dark:text-gray-300"
            aria-label="Account type"
          >
            {PANEL_ORDER.map((t) => (
              <option key={t} value={t}>{PANEL_LABELS[t]}</option>
            ))}
          </select>
        )}
      </div>
    </div>
  )
}

// --- Droppable Group (accepts account drops) ---

function DroppableGroup({
  group,
  panelType,
  isMobile,
  children,
}: {
  group: GroupItem
  panelType: PanelType
  isMobile: boolean
  children: React.ReactNode
}) {
  const { ref: droppableRef } = useDroppable({ id: `group-${group.id}` })
  const { ref: draggableRef, isDragging } = useDraggable({
    id: `drag-group-${group.id}`,
    data: { panelType, isGroup: true, groupId: group.id },
    disabled: isMobile,
  })

  return (
    <div
      ref={(node) => { droppableRef(node); draggableRef(node) }}
      data-testid={`group-${group.id}`}
      className={`rounded-md border-2 border-dashed border-transparent transition-colors ${isDragging ? 'opacity-30' : ''}`}
    >
      {children}
    </div>
  )
}

// --- Draggable Group Member ---

function DraggableGroupMember({
  member,
  groupId,
  isMobile,
}: {
  member: { id: string; name: string; original_name: string; balance: string; org_name: string; display_name?: string | null }
  groupId: number
  isMobile: boolean
}) {
  const { ref, isDragging } = useDraggable({
    id: member.id,
    data: { groupId },
    disabled: isMobile,
  })

  return (
    <div
      ref={ref}
      className={`flex justify-between items-center text-sm py-1 px-3 ${isDragging ? 'opacity-30' : ''}`}
    >
      {!isMobile && (
        <span className="cursor-grab text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 shrink-0 mr-1" title="Drag to move">
          <GripIcon />
        </span>
      )}
      <span className="text-gray-600 dark:text-gray-300 truncate pr-2 flex-1">
        {getAccountDisplayName(member)}
      </span>
      <span className="text-gray-700 dark:text-gray-300 font-medium shrink-0">
        {formatCurrency(member.balance)}
      </span>
    </div>
  )
}

// --- Account Group (droppable) ---

function TrashIcon() {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M3 6h18" /><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6" />
      <path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2" />
    </svg>
  )
}

function AccountGroup({
  panelType,
  accounts,
  editingId,
  editValue,
  savingId,
  errorId,
  isMobile,
  onStartEdit,
  onEditChange,
  onSaveEdit,
  onCancelEdit,
  onReset,
  onHide,
  onTypeChange,
  groups,
  expandedGroupIds,
  onToggleGroup,
  deletingGroupId,
  onRequestDeleteGroup,
  onConfirmDeleteGroup,
  onCancelDeleteGroup,
}: {
  panelType: PanelType
  accounts: AccountItem[]
  editingId: string | null
  editValue: string
  savingId: string | null
  errorId: string | null
  isMobile: boolean
  onStartEdit: (id: string, currentName: string) => void
  onEditChange: (value: string) => void
  onSaveEdit: (id: string) => void
  onCancelEdit: () => void
  onReset: (id: string) => void
  onHide: (id: string) => void
  onTypeChange: (id: string, newType: PanelType) => void
  groups?: GroupItem[]
  expandedGroupIds?: Set<number>
  onToggleGroup?: (id: number) => void
  deletingGroupId?: number | null
  onRequestDeleteGroup?: (id: number) => void
  onConfirmDeleteGroup?: (id: number) => void
  onCancelDeleteGroup?: () => void
}) {
  const { ref } = useDroppable({ id: panelType })
  const totalCount = accounts.length + (groups?.reduce((sum, g) => sum + g.members.length, 0) ?? 0)

  return (
    <div ref={ref} data-testid={`account-group-${panelType}`}>
      <h3 className="text-sm font-semibold text-gray-700 dark:text-gray-300 uppercase tracking-wide mb-1 flex items-center gap-2">
        {PANEL_LABELS[panelType]}
        <span className="text-xs font-normal text-gray-400">({totalCount})</span>
      </h3>

      {/* Groups */}
      {groups && groups.length > 0 && (
        <div className="space-y-1 mb-1">
          {groups.map((group) => {
            const isExpanded = expandedGroupIds?.has(group.id) ?? false
            const isDeleting = deletingGroupId === group.id
            return (
              <DroppableGroup key={group.id} group={group} panelType={panelType} isMobile={isMobile}>
                <div className="flex items-center gap-1 py-1 px-3 rounded-md hover:bg-gray-50 dark:hover:bg-gray-800">
                  {!isMobile && (
                    <span className="cursor-grab text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 shrink-0" title="Drag to move group">
                      <GripIcon />
                    </span>
                  )}
                  <button
                    type="button"
                    onClick={() => onToggleGroup?.(group.id)}
                    className="flex items-center gap-1 flex-1 min-w-0 text-left"
                    aria-expanded={isExpanded}
                  >
                    <ChevronIcon expanded={isExpanded} />
                    <span className="text-sm font-semibold text-gray-900 dark:text-gray-100 truncate">{group.name}</span>
                    <span className="text-xs text-gray-500 dark:text-gray-400 shrink-0">({group.members.length} accounts)</span>
                  </button>
                  <span className="text-sm font-medium text-gray-700 dark:text-gray-300 shrink-0">
                    {formatCurrency(group.total_balance)}
                  </span>
                  <button
                    type="button"
                    onClick={() => onRequestDeleteGroup?.(group.id)}
                    className="p-1 text-gray-400 hover:text-red-500 dark:hover:text-red-400 shrink-0"
                    title="Delete group"
                    aria-label="Delete group"
                  >
                    <TrashIcon />
                  </button>
                </div>

                {/* Delete confirmation */}
                {isDeleting && (
                  <div className="mx-3 mb-2 p-2 bg-gray-50 dark:bg-gray-800 rounded text-sm">
                    <p className="text-gray-700 dark:text-gray-300 mb-2">
                      Delete &quot;{group.name}&quot;? Member accounts will be ungrouped, not deleted.
                    </p>
                    <div className="flex gap-2">
                      <button
                        type="button"
                        onClick={() => onCancelDeleteGroup?.()}
                        className="px-2 py-1 text-xs rounded bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300"
                      >
                        Keep Group
                      </button>
                      <button
                        type="button"
                        onClick={() => onConfirmDeleteGroup?.(group.id)}
                        className="px-2 py-1 text-xs rounded bg-red-100 dark:bg-red-900 text-red-600 dark:text-red-400"
                      >
                        Delete Group
                      </button>
                    </div>
                  </div>
                )}

                {/* Expanded members */}
                {isExpanded && (
                  <div className="pl-8 space-y-0.5">
                    {group.members.map((member) => (
                      <DraggableGroupMember key={member.id} member={member} groupId={group.id} isMobile={isMobile} />
                    ))}
                  </div>
                )}
              </DroppableGroup>
            )
          })}
        </div>
      )}

      {accounts.length === 0 && (!groups || groups.length === 0) ? (
        <p className="text-xs text-gray-400 dark:text-gray-500 py-2 px-3">No accounts</p>
      ) : (
        <div className="space-y-0.5">
          {accounts.map((account) => (
            <DraggableAccountRow
              key={account.id}
              account={account}
              panelType={panelType}
              editingId={editingId}
              editValue={editValue}
              savingId={savingId}
              errorId={errorId}
              isMobile={isMobile}
              onStartEdit={onStartEdit}
              onEditChange={onEditChange}
              onSaveEdit={onSaveEdit}
              onCancelEdit={onCancelEdit}
              onReset={onReset}
              onHide={onHide}
              onTypeChange={onTypeChange}
            />
          ))}
        </div>
      )}
    </div>
  )
}

// --- Main AccountsSection ---

export default function AccountsSection({ onAccountRestored }: AccountsSectionProps) {
  const [accounts, setAccounts] = useState<AccountsResponse | null>(null)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editValue, setEditValue] = useState('')
  const [savingId, setSavingId] = useState<string | null>(null)
  const [errorId, setErrorId] = useState<string | null>(null)
  const [hiddenExpanded, setHiddenExpanded] = useState(false)
  const [isMobile, setIsMobile] = useState(false)
  const [loading, setLoading] = useState(true)
  const [creatingGroup, setCreatingGroup] = useState(false)
  const [newGroupName, setNewGroupName] = useState('')
  const [expandedGroupIds, setExpandedGroupIds] = useState<Set<number>>(new Set())
  const [deletingGroupId, setDeletingGroupId] = useState<number | null>(null)

  // Suppress unused variable warning - onAccountRestored is called by parent
  void onAccountRestored

  // Detect mobile viewport
  useEffect(() => {
    const mq = window.matchMedia('(max-width: 768px)')
    setIsMobile(mq.matches)
    function handler(e: MediaQueryListEvent) {
      setIsMobile(e.matches)
    }
    mq.addEventListener('change', handler)
    return () => mq.removeEventListener('change', handler)
  }, [])

  // Load accounts on mount
  useEffect(() => {
    loadAccounts()
  }, [])

  async function loadAccounts() {
    try {
      const data = await getAccounts(true)
      setAccounts(data)
    } catch {
      // Silently fail; component just won't render accounts
    } finally {
      setLoading(false)
    }
  }

  // Collect all visible and hidden accounts
  const visibleAccounts: Record<PanelType, AccountItem[]> = {
    liquid: [],
    savings: [],
    investments: [],
    other: [],
  }
  const hiddenAccounts: AccountItem[] = []

  if (accounts) {
    for (const panel of PANEL_ORDER) {
      for (const acct of accounts[panel]) {
        if (acct.hidden_at) {
          hiddenAccounts.push(acct)
        } else {
          visibleAccounts[panel].push(acct)
        }
      }
    }
  }

  // --- Handlers ---

  function handleStartEdit(id: string, currentName: string) {
    setEditingId(id)
    setEditValue(currentName)
    setErrorId(null)
  }

  function handleEditChange(value: string) {
    setEditValue(value)
  }

  const handleSaveEdit = useCallback(async (id: string) => {
    if (editingId !== id) return
    const trimmed = editValue.trim()
    setEditingId(null)

    // If name is empty, clear display_name (reset to original)
    const data: UpdateAccountRequest = {
      display_name: trimmed || null,
    }

    setSavingId(id)
    setErrorId(null)
    try {
      const updated = await updateAccount(id, data)
      // Optimistic update: replace the account in local state
      setAccounts((prev) => {
        if (!prev) return prev
        return replaceAccountInResponse(prev, updated)
      })
    } catch {
      setErrorId(id)
    } finally {
      setSavingId(null)
    }
  }, [editingId, editValue])

  function handleCancelEdit() {
    setEditingId(null)
    setEditValue('')
    setErrorId(null)
  }

  async function handleReset(id: string) {
    setSavingId(id)
    setErrorId(null)
    try {
      const updated = await updateAccount(id, { display_name: null })
      setAccounts((prev) => {
        if (!prev) return prev
        return replaceAccountInResponse(prev, updated)
      })
    } catch {
      setErrorId(id)
    } finally {
      setSavingId(null)
    }
  }

  async function handleHide(id: string) {
    setSavingId(id)
    setErrorId(null)
    try {
      const updated = await updateAccount(id, { hidden: true })
      setAccounts((prev) => {
        if (!prev) return prev
        return replaceAccountInResponse(prev, updated)
      })
    } catch {
      setErrorId(id)
    } finally {
      setSavingId(null)
    }
  }

  async function handleUnhide(id: string) {
    setSavingId(id)
    setErrorId(null)
    try {
      const updated = await updateAccount(id, { hidden: false })
      setAccounts((prev) => {
        if (!prev) return prev
        return replaceAccountInResponse(prev, updated)
      })
    } catch {
      setErrorId(id)
    } finally {
      setSavingId(null)
    }
  }

  async function handleTypeChange(id: string, newType: PanelType) {
    setSavingId(id)
    setErrorId(null)

    // Optimistic: move account to new group immediately to avoid animation glitch
    setAccounts((prev) => {
      if (!prev) return prev
      let movedAccount: AccountItem | null = null
      const result: AccountsResponse = { liquid: [], savings: [], investments: [], other: [] }
      for (const panel of PANEL_ORDER) {
        result[panel] = prev[panel].filter((a) => {
          if (a.id === id) {
            movedAccount = a
            return false
          }
          return true
        })
      }
      if (movedAccount) {
        result[newType] = [...result[newType], movedAccount]
      }
      return result
    })

    try {
      const overrideValue = PANEL_TYPE_TO_OVERRIDE[newType]
      const updated = await updateAccount(id, { account_type_override: overrideValue })
      // Sync server response into the optimistically-moved position
      setAccounts((prev) => {
        if (!prev) return prev
        const result: AccountsResponse = { liquid: [], savings: [], investments: [], other: [] }
        for (const panel of PANEL_ORDER) {
          result[panel] = prev[panel].map((a) => (a.id === updated.id ? updated : a))
        }
        return result
      })
    } catch {
      setErrorId(id)
      // Rollback: re-fetch from server
      await loadAccounts()
    } finally {
      setSavingId(null)
    }
  }

  async function handleCreateGroup(panelType: PanelType) {
    const trimmed = newGroupName.trim()
    if (!trimmed) return
    try {
      await createGroup(trimmed, PANEL_TYPE_TO_OVERRIDE[panelType])
      setCreatingGroup(false)
      setNewGroupName('')
      await loadAccounts()
    } catch {
      // keep input open on error
    }
  }

  async function handleAddGroupMember(groupId: number, accountId: string) {
    try {
      await addGroupMember(groupId, accountId)
      await loadAccounts()
    } catch {
      // silently fail
    }
  }

  async function handleRemoveGroupMember(groupId: number, accountId: string) {
    try {
      await removeGroupMember(groupId, accountId)
      await loadAccounts()
    } catch {
      // silently fail
    }
  }

  async function handleDeleteGroup(groupId: number) {
    setDeletingGroupId(null)
    try {
      await deleteGroupApi(groupId)
      await loadAccounts()
    } catch {
      // silently fail
    }
  }

  async function handleGroupPanelChange(groupId: number, newPanelType: PanelType) {
    try {
      await updateGroup(groupId, { panel_type: PANEL_TYPE_TO_OVERRIDE[newPanelType] })
      await loadAccounts()
    } catch {
      // silently fail
    }
  }

  function toggleGroupExpanded(id: number) {
    setExpandedGroupIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  function handleDragEnd(event: any) {
    const { source, target } = event.operation ?? {}
    if (!source || !target) return

    const sourceId = String(source.id)
    const sourcePanelType = source.data?.panelType as PanelType | undefined
    const sourceGroupId = source.data?.groupId as number | undefined
    const isGroupDrag = source.data?.isGroup === true
    const targetId = String(target.id)

    // Dragging a whole group to a different panel
    if (isGroupDrag && sourceGroupId) {
      if (PANEL_ORDER.includes(targetId as PanelType) && sourcePanelType !== targetId) {
        handleGroupPanelChange(sourceGroupId, targetId as PanelType)
      }
      return
    }

    // Drop an account onto a group droppable (target id starts with "group-")
    if (targetId.startsWith('group-')) {
      const targetGroupId = Number(targetId.replace('group-', ''))
      if (sourceGroupId && sourceGroupId !== targetGroupId) {
        // Move between groups
        handleRemoveGroupMember(sourceGroupId, sourceId).then(() => handleAddGroupMember(targetGroupId, sourceId))
      } else if (!sourceGroupId) {
        // Add standalone account to group
        handleAddGroupMember(targetGroupId, sourceId)
      }
      return
    }

    // Drop onto a panel droppable
    if (!PANEL_ORDER.includes(targetId as PanelType)) return
    const targetPanelType = targetId as PanelType

    if (sourceGroupId) {
      // Remove from group (ungroup) by dropping on panel
      handleRemoveGroupMember(sourceGroupId, sourceId)
      return
    }

    if (sourcePanelType && sourcePanelType !== targetPanelType) {
      handleTypeChange(sourceId, targetPanelType)
    }
  }

  if (loading) {
    return (
      <div className="bg-white dark:bg-gray-900 rounded-xl shadow-md p-6">
        <h2 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">Accounts</h2>
        <p className="text-sm text-gray-500 dark:text-gray-400">Loading accounts...</p>
      </div>
    )
  }

  if (!accounts) return null

  const totalVisible = PANEL_ORDER.reduce((sum, p) => sum + visibleAccounts[p].length, 0)
  const totalHidden = hiddenAccounts.length

  if (totalVisible === 0 && totalHidden === 0) return null

  return (
    <div className="bg-white dark:bg-gray-900 rounded-xl shadow-md p-6" data-testid="accounts-section">
      <div className="flex justify-between items-center mb-4">
        <h2 className="text-lg font-medium text-gray-900 dark:text-gray-100">Accounts</h2>
        <button
          type="button"
          onClick={() => setCreatingGroup(true)}
          className="text-sm font-semibold text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 transition-colors cursor-pointer"
        >
          + New Group
        </button>
      </div>

      {creatingGroup && (
        <div className="mb-4">
          <input
            type="text"
            value={newGroupName}
            onChange={(e) => setNewGroupName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && newGroupName.trim()) handleCreateGroup('liquid')
              if (e.key === 'Escape') { setCreatingGroup(false); setNewGroupName('') }
            }}
            autoFocus
            placeholder="Group name"
            className="text-sm border border-blue-400 rounded px-2 py-1 focus:outline-none focus:ring-2 focus:ring-blue-500 w-full bg-white dark:bg-gray-800 text-gray-900 dark:text-gray-100"
          />
        </div>
      )}

      <DragDropProvider onDragEnd={handleDragEnd}>
        <div className="space-y-4">
          {PANEL_ORDER.map((panelType) => {
            const panelAccounts = visibleAccounts[panelType]
            const panelGroups = accounts?.groups?.filter(
              (g) => g.panel_type === PANEL_TYPE_TO_OVERRIDE[panelType]
            ) ?? []
            // Only show groups that have accounts or are receiving drops
            if (panelAccounts.length === 0 && panelGroups.length === 0 && panelType === 'other') return null
            return (
              <div key={panelType}>
                <AccountGroup
                  panelType={panelType}
                  accounts={panelAccounts}
                  editingId={editingId}
                  editValue={editValue}
                  savingId={savingId}
                  errorId={errorId}
                  isMobile={isMobile}
                  onStartEdit={handleStartEdit}
                  onEditChange={handleEditChange}
                  onSaveEdit={handleSaveEdit}
                  onCancelEdit={handleCancelEdit}
                  onReset={handleReset}
                  onHide={handleHide}
                  onTypeChange={handleTypeChange}
                  groups={panelGroups}
                  expandedGroupIds={expandedGroupIds}
                  onToggleGroup={toggleGroupExpanded}
                  deletingGroupId={deletingGroupId}
                  onRequestDeleteGroup={setDeletingGroupId}
                  onConfirmDeleteGroup={handleDeleteGroup}
                  onCancelDeleteGroup={() => setDeletingGroupId(null)}
                />
              </div>
            )
          })}
        </div>
      </DragDropProvider>

      {/* Hidden accounts collapsible */}
      {totalHidden > 0 && (
        <div className="mt-6 border-t border-gray-200 dark:border-gray-700 pt-4">
          <button
            type="button"
            onClick={() => setHiddenExpanded(!hiddenExpanded)}
            className="flex items-center gap-2 text-sm font-medium text-gray-600 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-200"
            aria-expanded={hiddenExpanded}
          >
            <ChevronIcon expanded={hiddenExpanded} />
            Hidden Accounts ({totalHidden})
          </button>

          {hiddenExpanded && (
            <div className="mt-2 space-y-1" data-testid="hidden-accounts">
              {hiddenAccounts.map((account) => (
                <div
                  key={account.id}
                  className="flex items-center gap-2 py-2 px-3 rounded-md opacity-50 text-gray-400 dark:text-gray-600"
                  data-testid={`hidden-account-${account.id}`}
                >
                  <div className="flex-1 min-w-0">
                    <span className="text-sm truncate block">
                      {getAccountDisplayName(account)}
                    </span>
                  </div>
                  <span className="text-sm font-medium shrink-0">
                    {formatCurrency(account.balance)}
                  </span>
                  <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200">
                    Hidden
                  </span>
                  <button
                    type="button"
                    onClick={() => handleUnhide(account.id)}
                    className="p-1 text-blue-500 hover:text-blue-700 dark:text-blue-400 dark:hover:text-blue-300"
                    title="Unhide account"
                    aria-label="Unhide account"
                  >
                    <EyeIcon />
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

// Helper: replace a single account in the AccountsResponse by ID
function replaceAccountInResponse(
  response: AccountsResponse,
  updated: AccountItem
): AccountsResponse {
  const result: AccountsResponse = {
    liquid: [],
    savings: [],
    investments: [],
    other: [],
  }

  for (const panel of PANEL_ORDER) {
    result[panel] = response[panel].map((a) =>
      a.id === updated.id ? updated : a
    )
  }

  return result
}
