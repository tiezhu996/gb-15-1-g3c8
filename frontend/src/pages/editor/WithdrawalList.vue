<template>
  <el-card shadow="never">
    <template #header>
      <div class="row-between">
        <span>撤稿申请审批（{{ pagination.total.value }}）</span>
        <el-select v-model="status" placeholder="全部状态" style="width: 160px" @change="onStatusChange">
          <el-option label="全部状态" value="" />
          <el-option v-for="(text, key) in WITHDRAWAL_STATUS_MAP" :key="key" :label="text" :value="key" />
        </el-select>
      </div>
    </template>
    <el-table :data="pagination.items.value" v-loading="pagination.loading.value" stripe>
      <el-table-column label="论文" min-width="200" show-overflow-tooltip>
        <template #default="{ row }">{{ row.paper?.title || `论文 #${row.paper_id}` }}</template>
      </el-table-column>
      <el-table-column label="申请人" width="110">
        <template #default="{ row }">{{ row.applicant?.real_name || row.applicant?.username || '-' }}</template>
      </el-table-column>
      <el-table-column prop="reason" label="撤稿原因" min-width="160" show-overflow-tooltip />
      <el-table-column label="替代处理说明" min-width="150" show-overflow-tooltip>
        <template #default="{ row }">{{ row.alt_handling_note || '-' }}</template>
      </el-table-column>
      <el-table-column label="状态" width="90">
        <template #default="{ row }"><StatusBadge :status="row.status" kind="withdrawal" /></template>
      </el-table-column>
      <el-table-column label="处理结果" min-width="140" show-overflow-tooltip>
        <template #default="{ row }">{{ row.process_result || '-' }}</template>
      </el-table-column>
      <el-table-column label="申请时间" width="150">
        <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="150" fixed="right">
        <template #default="{ row }">
          <el-button v-if="row.status === 'pending'" link type="primary" @click="openProcess(row)">处理</el-button>
          <el-button link type="primary" @click="router.push(`/editor/papers/${row.paper_id}`)">论文</el-button>
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
      @current-change="() => pagination.load({ status: status })"
      class="pager"
    />
  </el-card>

  <el-dialog v-model="dialog.visible" title="处理撤稿申请" width="560px">
    <template v-if="dialog.item">
      <el-descriptions :column="1" border class="mb">
        <el-descriptions-item label="论文">{{ dialog.item.paper?.title || `论文 #${dialog.item.paper_id}` }}</el-descriptions-item>
        <el-descriptions-item label="撤稿原因">{{ dialog.item.reason }}</el-descriptions-item>
        <el-descriptions-item label="替代处理说明">{{ dialog.item.alt_handling_note || '（未填写）' }}</el-descriptions-item>
      </el-descriptions>
      <el-alert
        title="批准后论文进入已撤稿状态，将从论文库与统计中排除，未完成审稿一并关闭；驳回则恢复原流程"
        type="warning"
        :closable="false"
        class="mb"
      />
      <el-form :model="dialog" label-width="90px">
        <el-form-item label="处理结论">
          <el-radio-group v-model="dialog.approve">
            <el-radio :value="true">批准撤稿</el-radio>
            <el-radio :value="false">驳回申请</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="处理结果" required>
          <el-input v-model="dialog.result" type="textarea" :rows="3" placeholder="处理意见将展示给作者（至少 2 字）" />
        </el-form-item>
      </el-form>
    </template>
    <template #footer>
      <el-button @click="dialog.visible = false">取消</el-button>
      <el-button type="primary" :loading="dialog.loading" @click="submitProcess">提交处理</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { WITHDRAWAL_STATUS_MAP } from '../../constants'
import { listWithdrawals, processWithdrawal } from '../../api/withdrawal'
import type { WithdrawalItem } from '../../api/types'
import EmptyState from '../../components/EmptyState.vue'
import StatusBadge from '../../components/StatusBadge.vue'
import { usePagination } from '../../hooks/usePagination'
import { formatTime } from '../../utils/format'

const router = useRouter()
const status = ref('')
const pagination = usePagination<WithdrawalItem>((params) => listWithdrawals({ ...params, status: status.value }))
const dialog = reactive({
  visible: false,
  item: null as WithdrawalItem | null,
  approve: true,
  result: '',
  loading: false
})

function onStatusChange() {
  pagination.load({ status: status.value })
}

function openProcess(row: WithdrawalItem) {
  dialog.item = row
  dialog.approve = true
  dialog.result = ''
  dialog.visible = true
}

async function submitProcess() {
  if (!dialog.item) return
  if (dialog.result.trim().length < 2) {
    ElMessage.warning('请填写处理结果（至少 2 字）')
    return
  }
  dialog.loading = true
  try {
    await processWithdrawal(dialog.item.id, { approve: dialog.approve, process_result: dialog.result })
    ElMessage.success(dialog.approve ? '已批准撤稿，论文进入已撤稿状态' : '已驳回，论文恢复原流程')
    dialog.visible = false
    pagination.load({ status: status.value })
  } catch {
    // 拦截器已提示
  } finally {
    dialog.loading = false
  }
}

onMounted(() => pagination.load({ status: '' }))
</script>

<style scoped>
.mb {
  margin-bottom: 12px;
}
</style>
