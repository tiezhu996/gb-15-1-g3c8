<template>
  <el-card shadow="never">
    <template #header>
      <div class="row-between">
        <span>我的投稿（{{ pagination.total.value }}）</span>
        <el-button type="primary" @click="router.push('/papers/create')">新建投稿</el-button>
      </div>
    </template>
    <el-table :data="pagination.items.value" v-loading="pagination.loading.value" stripe>
      <el-table-column prop="title" label="标题" min-width="220" show-overflow-tooltip />
      <el-table-column label="学科" width="110">
        <template #default="{ row }">{{ subjectText(row.subject) }}</template>
      </el-table-column>
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <StatusBadge :status="row.status" kind="paper" />
        </template>
      </el-table-column>
      <el-table-column label="撤稿申请" width="100">
        <template #default="{ row }">
          <StatusBadge v-if="withdrawalMap[row.id]" :status="withdrawalMap[row.id].status" kind="withdrawal" />
          <span v-else>-</span>
        </template>
      </el-table-column>
      <el-table-column label="相似度" width="90">
        <template #default="{ row }">
          <span :class="{ 'high-similarity': row.similarity > 30 }">{{ formatPercent(row.similarity) }}</span>
        </template>
      </el-table-column>
      <el-table-column label="版本" width="60">
        <template #default="{ row }">V{{ row.version }}</template>
      </el-table-column>
      <el-table-column label="投稿时间" width="150">
        <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="router.push(`/papers/${row.id}`)">详情</el-button>
          <el-button
            v-if="row.status === 'revision' && withdrawalMap[row.id]?.status !== 'pending'"
            link
            type="warning"
            @click="router.push(`/papers/${row.id}/revise`)"
          >
            修改
          </el-button>
          <el-button v-if="canWithdraw(row)" link type="danger" @click="openWithdraw(row)">撤稿</el-button>
        </template>
      </el-table-column>
    </el-table>
    <EmptyState
      v-if="pagination.total.value === 0 && !pagination.loading.value"
      description="还没有投稿，点击右上角新建投稿"
    />
    <el-pagination
      v-model:current-page="pagination.page.value"
      v-model:page-size="pagination.size.value"
      :total="pagination.total.value"
      layout="total, prev, pager, next"
      @current-change="() => pagination.load()"
      class="pager"
    />
  </el-card>

  <el-dialog v-model="dialog.visible" title="申请撤稿" width="560px">
    <el-alert
      v-if="dialog.paper"
      :title="`论文「${dialog.paper.title}」当前状态：${paperStatusText(dialog.paper.status)}`"
      type="warning"
      :closable="false"
      class="mb"
    />
    <el-alert
      v-if="needAltNote"
      title="论文处于外审或修稿阶段，须填写替代处理说明"
      type="error"
      :closable="false"
      class="mb"
    />
    <el-form :model="dialog" label-width="110px">
      <el-form-item label="撤稿原因" required>
        <el-input v-model="dialog.reason" type="textarea" :rows="3" placeholder="请说明撤稿原因（至少 5 字）" />
      </el-form-item>
      <el-form-item v-if="needAltNote" label="替代处理说明" required>
        <el-input
          v-model="dialog.altNote"
          type="textarea"
          :rows="3"
          placeholder="外审/修稿中的稿件如何处理，如：已通知审稿人停止审稿、改投其他期刊"
        />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dialog.visible = false">取消</el-button>
      <el-button type="danger" :loading="dialog.loading" @click="submitWithdraw">确认撤稿</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { listMyPapers } from '../../api/paper'
import { applyWithdrawal, listMyWithdrawals } from '../../api/withdrawal'
import type { Paper, WithdrawalItem } from '../../api/types'
import EmptyState from '../../components/EmptyState.vue'
import StatusBadge from '../../components/StatusBadge.vue'
import { usePagination } from '../../hooks/usePagination'
import { formatPercent, formatTime, paperStatusText, subjectText } from '../../utils/format'

const router = useRouter()
const pagination = usePagination<Paper>((params) => listMyPapers(params))
const withdrawals = ref<WithdrawalItem[]>([])
const dialog = reactive({
  visible: false,
  paper: null as Paper | null,
  reason: '',
  altNote: '',
  loading: false
})

// 每篇论文的最新撤稿申请（列表刷新后显示处理状态）
const withdrawalMap = computed(() => {
  const map: Record<number, WithdrawalItem> = {}
  for (const w of withdrawals.value) {
    if (!map[w.paper_id]) map[w.paper_id] = w
  }
  return map
})

const needAltNote = computed(
  () => dialog.paper && ['external_review', 'revision'].includes(dialog.paper.status)
)

function canWithdraw(row: Paper) {
  return ['submitted', 'initial_review', 'external_review', 'revision'].includes(row.status) &&
    withdrawalMap.value[row.id]?.status !== 'pending'
}

function openWithdraw(row: Paper) {
  dialog.paper = row
  dialog.reason = ''
  dialog.altNote = ''
  dialog.visible = true
}

async function loadWithdrawals() {
  try {
    const res = await listMyWithdrawals({ page: 1, size: 100 })
    withdrawals.value = res.items
  } catch {
    // 拦截器已提示
  }
}

async function submitWithdraw() {
  if (!dialog.paper) return
  if (dialog.reason.trim().length < 5) {
    ElMessage.warning('撤稿原因至少 5 字')
    return
  }
  if (needAltNote.value && !dialog.altNote.trim()) {
    ElMessage.warning('外审或修稿中的论文须填写替代处理说明')
    return
  }
  dialog.loading = true
  try {
    await applyWithdrawal(dialog.paper.id, {
      reason: dialog.reason,
      alt_handling_note: dialog.altNote
    })
    ElMessage.success('撤稿申请已提交，待编辑处理')
    dialog.visible = false
    await Promise.all([pagination.load(), loadWithdrawals()])
  } catch {
    // 拦截器已提示
  } finally {
    dialog.loading = false
  }
}

onMounted(() => {
  pagination.load()
  loadWithdrawals()
})
</script>

<style scoped>
.mb {
  margin-bottom: 12px;
}
</style>
