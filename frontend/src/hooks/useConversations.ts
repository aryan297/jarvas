import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { chatService } from '@/services/chat.service'
import toast from 'react-hot-toast'

export function useConversations() {
  return useQuery({
    queryKey: ['conversations'],
    queryFn: () => chatService.listConversations(),
    select: (res) => res.data.data ?? [],
  })
}

export function useCreateConversation() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (title?: string) => chatService.createConversation({ title }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['conversations'] }),
    onError: () => toast.error('Failed to create conversation'),
  })
}

export function useDeleteConversation() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => chatService.deleteConversation(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['conversations'] }),
    onError: () => toast.error('Failed to delete conversation'),
  })
}
