<template>
  <el-card v-if="withdrawal" shadow="never" class="withdrawal-card" :class="{ pending: isPending }">
    <template #header>
      <div class="row-between">
        <span>撤稿申请</span>
        <StatusBadge :status="withdrawal.status" kind="withdrawal" />
      </div>
    </template>
    <el-descriptions :column="border ? 2 : 1" border>
      <el-descriptions-item label="申请人">
        {{ withdrawal.applicant?.real_name || withdrawal.applicant?.username || `用户 #${withdrawal.applicant_id}` }}
      </el-descriptions-item>
      <el-descriptions-item label="申请时间">{{ formatTime(withdrawal.created_at) }}</el-descriptions-item>
      <el-descriptions-item label="撤稿原因" :span="2">{{ withdrawal.reason }}</el-descriptions-item>
      <el-descriptions-item v-if="withdrawal.alternative_note" label="替代处理说明" :span="2">
        {{ withdrawal.alternative_note }}
      </el-descriptions-item>
      <template v-if="withdrawal.status !== 'pending'">
        <el-descriptions-item label="处理人">
          {{ withdrawal.processed_by?.real_name || withdrawal.processed_by?.username || '-' }}
        </el-descriptions-item>
        <el-descriptions-item label="处理时间">{{ formatTime(withdrawal.processed_at) }}</el-descriptions-item>
        <el-descriptions-item label="处理结果" :span="2">
          <StatusBadge :status="withdrawal.status" kind="withdrawal" style="margin-right: 8px" />
          <span>{{ withdrawal.decision_comment || '（未填写处理意见）' }}</span>
        </el-descriptions-item>
      </template>
    </el-descriptions>
    <el-alert
      v-if="isPending"
      class="mt-16"
      title="撤稿申请待编辑部处理，审稿、修稿与查重流程已暂停（记录仍可查看）"
      type="warning"
      :closable="false"
    />
    <el-alert
      v-else-if="withdrawal.status === 'rejected'"
      class="mt-16"
      title="撤稿申请已驳回，论文恢复原审稿流程"
      type="info"
      :closable="false"
    />
    <el-alert
      v-else
      class="mt-16"
      title="撤稿申请已批准，论文进入已撤稿终态，已从论文库与统计中排除"
      type="success"
      :closable="false"
    />
  </el-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { WithdrawalItem } from '../api/types'
import StatusBadge from './StatusBadge.vue'
import { formatTime } from '../utils/format'

const props = defineProps<{ withdrawal?: WithdrawalItem | null; border?: boolean }>()

const isPending = computed(() => props.withdrawal?.status === 'pending')
</script>

<style scoped>
.withdrawal-card.pending {
  border-color: var(--el-color-warning);
}
</style>
