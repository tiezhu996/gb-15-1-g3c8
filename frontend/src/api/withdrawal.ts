import request from '../utils/request'
import type { PageResult, WithdrawalItem } from './types'

// 作者对本人论文发起撤稿申请
export function applyWithdrawal(paperId: number | string, data: { reason: string; alternative_note?: string }) {
  return request.post(`/papers/${paperId}/withdrawal`, data) as Promise<WithdrawalItem>
}

// 论文最近一次撤稿申请（刷新后展示原因、处理结果与状态）
export function getPaperWithdrawal(paperId: number | string) {
  return request.get(`/papers/${paperId}/withdrawal`) as Promise<WithdrawalItem>
}

// 编辑撤稿处理队列
export function listWithdrawals(params: Record<string, unknown>) {
  return request.get('/withdrawals', { params }) as Promise<PageResult<WithdrawalItem>>
}

// 编辑批准/驳回撤稿申请
export function decideWithdrawal(id: number | string, data: { approve: boolean; comment?: string }) {
  return request.post(`/withdrawals/${id}/decision`, data) as Promise<WithdrawalItem>
}
