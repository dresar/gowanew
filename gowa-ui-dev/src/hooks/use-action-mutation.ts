import { useMutation } from '@tanstack/react-query'
import { toast } from 'sonner'
import { toApiError } from '@/lib/api-error'

/**
 * Wraps a mutation with the standard toast-on-success/error behaviour used by
 * every action form. Returns the mutation plus the last successful result for
 * inline display.
 */
export function useActionMutation<TData, TVars>(
  mutationFn: (vars: TVars) => Promise<TData>,
  options?: {
    successMessage?: string | ((data: TData) => string)
    onSuccess?: (data: TData, vars: TVars) => void
  },
) {
  return useMutation<TData, unknown, TVars>({
    mutationFn,
    onSuccess: (data, vars) => {
      const message =
        typeof options?.successMessage === 'function'
          ? options.successMessage(data)
          : (options?.successMessage ?? 'Done')
      toast.success(message)
      options?.onSuccess?.(data, vars)
    },
    onError: (error) => toast.error(toApiError(error).message),
  })
}
