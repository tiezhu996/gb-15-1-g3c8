<template>
  <el-card shadow="never">
    <template #header>
      <div class="row-between">
        <span>我的投稿（{{ pagination.total.value }}）</span>
        <el-button type="primary" @click="router.push('/papers/create')">新建投稿</el-button>
      </div>
    </template>
    <el-table :data="pagination.items.value" v-loading="pagination.loading.value" stripe>
      <el-table-column prop="title" label="标题" min-width="240" show-overflow-tooltip />
      <el-table-column label="学科" width="120">
        <template #default="{ row }">{{ subjectText(row.subject) }}</template>
      </el-table-column>
      <el-table-column label="状态" width="120">
        <template #default="{ row }">
          <StatusBadge :status="row.status" kind="paper" />
          <el-tooltip
            v-if="row.withdrawal?.status === 'pending'"
            content="撤稿申请待处理，流程已暂停"
            placement="top"
          >
            <StatusBadge status="pending" kind="withdrawal" style="margin-left: 4px" />
          </el-tooltip>
        </template>
      </el-table-column>
      <el-table-column label="相似度" width="100">
        <template #default="{ row }">
          <span :class="{ 'high-similarity': row.similarity > 30 }">{{ formatPercent(row.similarity) }}</span>
        </template>
      </el-table-column>
      <el-table-column label="版本" width="70">
        <template #default="{ row }">V{{ row.version }}</template>
      </el-table-column>
      <el-table-column label="投稿时间" width="150">
        <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="210" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="router.push(`/papers/${row.id}`)">详情</el-button>
          <el-button
            v-if="row.status === 'revision' && row.withdrawal?.status !== 'pending'"
            link
            type="warning"
            @click="router.push(`/papers/${row.id}/revise`)"
          >
            修改
          </el-button>
          <el-button
            v-if="canWithdraw(row)"
            link
            type="danger"
            @click="openWithdraw(row)"
          >
            撤稿
          </el-button>
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
      type="info"
      :closable="false"
      :title="`论文《${dialog.paper?.title || ''}》当前状态：${paperStatusText(dialog.paper?.status || '')}`"
      description="撤稿申请提交后须等待编辑部处理，期间审稿、修稿与查重流程暂停；批准后论文进入已撤稿终态，已录用论文不可撤稿。"
      class="mb"
    />
    <el-form ref="formRef" :model="dialog.form" :rules="rules" label-width="110px">
      <el-form-item label="撤稿原因" prop="reason">
        <el-input v-model="dialog.form.reason" type="textarea" :rows="3" placeholder="请填写撤稿原因（至少 10 字）" />
      </el-form-item>
      <el-form-item
        v-if="alternativeRequired"
        label="替代处理说明"
        prop="alternative_note"
      >
        <el-input
          v-model="dialog.form.alternative_note"
          type="textarea"
          :rows="3"
          placeholder="论文处于外审/修稿阶段，必须说明替代处理方式（如改投计划、后续安排，至少 10 字）"
        />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="dialog.visible = false">取消</el-button>
      <el-button type="danger" :loading="dialog.loading" @click="submitWithdraw">确认申请撤稿</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import type { FormInstance } from 'element-plus'
import { applyWithdrawal, listMyPapers } from '../../api/paper'
import type { Paper } from '../../api/types'
import EmptyState from '../../components/EmptyState.vue'
import StatusBadge from '../../components/StatusBadge.vue'
import { usePagination } from '../../hooks/usePagination'
import { formatPercent, formatTime, paperStatusText, subjectText } from '../../utils/format'

const router = useRouter()
const pagination = usePagination<Paper>((params) => listMyPapers(params))
const formRef = ref<FormInstance>()

const dialog = reactive({
  visible: false,
  loading: false,
  paper: null as Paper | null,
  form: { reason: '', alternative_note: '' }
})

// 未录用、非终态、无待处理撤稿申请时可发起一次撤稿。
function canWithdraw(row: Paper): boolean {
  return ['submitted', 'initial_review', 'external_review', 'revision'].includes(row.status) &&
    row.withdrawal?.status !== 'pending'
}

const alternativeRequired = computed(() =>
  ['external_review', 'revision'].includes(dialog.paper?.status || '')
)

const rules = computed(() => ({
  reason: [{ required: true, min: 10, message: '撤稿原因至少 10 字', trigger: 'blur' }],
  alternative_note: alternativeRequired.value
    ? [{ required: true, min: 10, message: '外审/修稿阶段必须填写替代处理说明（至少 10 字）', trigger: 'blur' }]
    : []
}))

function openWithdraw(row: Paper) {
  dialog.paper = row
  dialog.form = { reason: '', alternative_note: '' }
  dialog.visible = true
}

async function submitWithdraw() {
  if (!dialog.paper) return
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  dialog.loading = true
  try {
    await applyWithdrawal(dialog.paper.id, {
      reason: dialog.form.reason,
      alternative_note: alternativeRequired.value ? dialog.form.alternative_note : undefined
    })
    ElMessage.success('撤稿申请已提交，等待编辑部处理')
    dialog.visible = false
    pagination.load()
  } catch {
    // 拦截器已提示
  } finally {
    dialog.loading = false
  }
}

onMounted(() => pagination.load())
</script>
