import request from '../utils/request'
import type { PageResult, WithdrawalItem } from './types'

export function applyWithdrawal(paperId: number | string, data: { reason: string; alt_handling_note?: string }) {
  return request.post(`/papers/${paperId}/withdrawals`, data) as Promise<WithdrawalItem>
}

export function listMyWithdrawals(params: Record<string, unknown>) {
  return request.get('/withdrawals/mine', { params }) as Promise<PageResult<WithdrawalItem>>
}

export function listWithdrawals(params: Record<string, unknown>) {
  return request.get('/withdrawals', { params }) as Promise<PageResult<WithdrawalItem>>
}

export function listPaperWithdrawals(paperId: number | string) {
  return request.get(`/papers/${paperId}/withdrawals`) as Promise<WithdrawalItem[]>
}

export function processWithdrawal(id: number | string, data: { approve: boolean; process_result: string }) {
  return request.post(`/withdrawals/${id}/process`, data) as Promise<WithdrawalItem>
}
