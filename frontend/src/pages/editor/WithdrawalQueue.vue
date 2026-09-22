<template>
  <el-card shadow="never">
    <template #header>
      <div class="row-between">
        <span>撤稿申请队列（待处理 {{ pagination.total.value }}）</span>
        <el-select v-model="status" placeholder="待处理" style="width: 160px" @change="onStatusChange">
          <el-option label="待处理" value="pending" />
          <el-option label="已批准" value="approved" />
          <el-option label="已驳回" value="rejected" />
          <el-option label="全部" value="" />
        </el-select>
      </div>
    </template>
    <el-table :data="pagination.items.value" v-loading="pagination.loading.value" stripe>
      <el-table-column label="论文" min-width="220" show-overflow-tooltip>
        <template #default="{ row }">{{ row.paper?.title || `论文 #${row.paper_id}` }}</template>
      </el-table-column>
      <el-table-column label="申请人" width="120">
        <template #default="{ row }">{{ row.applicant?.real_name || row.applicant?.username || '-' }}</template>
      </el-table-column>
      <el-table-column label="撤稿原因" min-width="200" show-overflow-tooltip>
        <template #default="{ row }">{{ row.reason }}</template>
      </el-table-column>
      <el-table-column label="替代处理说明" min-width="200" show-overflow-tooltip>
        <template #default="{ row }">{{ row.alternative_note || '-' }}</template>
      </el-table-column>
      <el-table-column label="状态" width="100">
        <template #default="{ row }"><StatusBadge :status="row.status" kind="withdrawal" /></template>
      </el-table-column>
      <el-table-column label="申请时间" width="150">
        <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="150" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="router.push(`/editor/papers/${row.paper_id}`)">查看论文</el-button>
          <el-button v-if="row.status === 'pending'" link type="warning" @click="openDecide(row)">处理</el-button>
        </template>
      </el-table-column>
    </el-table>
    <EmptyState
      v-if="pagination.total.value === 0 && !pagination.loading.value"
      description="暂无撤稿申请"
    />
    <el-pagination
      v-model:current-page="pagination.page.value"
      v-model:page-size="pagination.size.value"
      :total="pagination.total.value"
      layout="total, prev, pager, next"
      @current-change="() => pagination.load({ status })"
      class="pager"
    />
  </el-card>

  <el-dialog v-model="dialog.visible" title="处理撤稿申请" width="560px">
    <el-descriptions :column="1" border>
      <el-descriptions-item label="论文">{{ dialog.item?.paper?.title }}</el-descriptions-item>
      <el-descriptions-item label="撤稿原因">{{ dialog.item?.reason }}</el-descriptions-item>
      <el-descriptions-item v-if="dialog.item?.alternative_note" label="替代处理说明">
        {{ dialog.item.alternative_note }}
      </el-descriptions-item>
    </el-descriptions>
    <el-radio-group v-model="dialog.approve" class="mt-16">
      <el-radio :value="true">批准撤稿（论文进入已撤稿，关闭未完成审稿，从论文库与统计排除）</el-radio>
      <el-radio :value="false">驳回（论文恢复原审稿流程）</el-radio>
    </el-radio-group>
    <el-input
      v-model="dialog.comment"
      class="mt-16"
      type="textarea"
      :rows="3"
      :placeholder="dialog.approve ? '处理意见（选填）' : '驳回必须填写处理意见，说明恢复流程的原因'"
    />
    <template #footer>
      <el-button @click="dialog.visible = false">取消</el-button>
      <el-button :type="dialog.approve ? 'danger' : 'primary'" :loading="dialog.loading" @click="submitDecide">
        确认{{ dialog.approve ? '批准撤稿' : '驳回' }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { listWithdrawals, decideWithdrawal } from '../../api/withdrawal'
import type { WithdrawalItem } from '../../api/types'
import EmptyState from '../../components/EmptyState.vue'
import StatusBadge from '../../components/StatusBadge.vue'
import { usePagination } from '../../hooks/usePagination'
import { formatTime } from '../../utils/format'

const router = useRouter()
const status = ref('pending')
const pagination = usePagination<WithdrawalItem>((params) =>
  listWithdrawals({ ...params, status: status.value })
)

const dialog = reactive({
  visible: false,
  loading: false,
  item: null as WithdrawalItem | null,
  approve: true,
  comment: ''
})

function onStatusChange() {
  pagination.load({ status: status.value })
}

function openDecide(row: WithdrawalItem) {
  dialog.item = row
  dialog.approve = true
  dialog.comment = ''
  dialog.visible = true
}

async function submitDecide() {
  if (!dialog.item) return
  if (!dialog.approve && !dialog.comment.trim()) {
    ElMessage.warning('驳回撤稿申请必须填写处理意见')
    return
  }
  dialog.loading = true
  try {
    await decideWithdrawal(dialog.item.id, { approve: dialog.approve, comment: dialog.comment })
    ElMessage.success(dialog.approve ? '已批准撤稿' : '已驳回，论文恢复原流程')
    dialog.visible = false
    pagination.load({ status: status.value })
  } catch {
    // 拦截器已提示
  } finally {
    dialog.loading = false
  }
}

onMounted(() => pagination.load({ status: 'pending' }))
</script>
