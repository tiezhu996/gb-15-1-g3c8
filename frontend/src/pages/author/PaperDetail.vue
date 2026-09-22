<template>
  <div v-loading="loading">
    <el-page-header class="mb" @back="router.back()">
      <template #content>
        <span style="font-weight: 600">投稿详情</span>
      </template>
    </el-page-header>
    <template v-if="paper">
      <el-card shadow="never" class="mt-16">
        <PaperStatusSteps :status="paper.status" />
      </el-card>
      <div class="mt-16">
        <PaperInfoCard :paper="paper" />
      </div>

      <el-card shadow="never" class="mt-16">
        <template #header>
          <div class="row-between">
            <span>撤稿申请</span>
            <el-button v-if="canWithdraw" type="danger" plain size="small" @click="withdrawVisible = true">
              申请撤稿
            </el-button>
          </div>
        </template>
        <EmptyState v-if="!withdrawals.length" description="暂无撤稿申请记录" />
        <el-table v-else :data="withdrawals" size="small" border>
          <el-table-column prop="reason" label="撤稿原因" min-width="180" show-overflow-tooltip />
          <el-table-column prop="alt_handling_note" label="替代处理说明" min-width="160" show-overflow-tooltip>
            <template #default="{ row }">{{ row.alt_handling_note || '-' }}</template>
          </el-table-column>
          <el-table-column label="状态" width="90">
            <template #default="{ row }"><StatusBadge :status="row.status" kind="withdrawal" /></template>
          </el-table-column>
          <el-table-column label="处理结果" min-width="160" show-overflow-tooltip>
            <template #default="{ row }">{{ row.process_result || '-' }}</template>
          </el-table-column>
          <el-table-column label="处理人" width="100">
            <template #default="{ row }">{{ row.processed_by?.real_name || row.processed_by?.username || '-' }}</template>
          </el-table-column>
          <el-table-column label="申请时间" width="150">
            <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
          </el-table-column>
          <el-table-column label="处理时间" width="150">
            <template #default="{ row }">{{ formatTime(row.processed_at) }}</template>
          </el-table-column>
        </el-table>
      </el-card>

      <el-card shadow="never" class="mt-16">
        <template #header>查重检测</template>
        <el-descriptions v-if="plagiarism" :column="3" border>
          <el-descriptions-item label="状态">
            <StatusBadge :status="plagiarism.status" kind="plagiarism" />
          </el-descriptions-item>
          <el-descriptions-item label="重复率">
            <span :class="{ 'high-similarity': plagiarism.similarity > 30 }">
              {{ formatPercent(plagiarism.similarity) }}
            </span>
          </el-descriptions-item>
          <el-descriptions-item label="检测时间">{{ formatTime(plagiarism.checked_at) }}</el-descriptions-item>
        </el-descriptions>
        <template v-if="reportItems.length">
          <el-divider content-position="left">重复段落标注</el-divider>
          <el-table :data="reportItems" size="small" border>
            <el-table-column prop="source" label="来源" width="160" />
            <el-table-column prop="paragraph" label="重复段落" />
            <el-table-column label="相似度" width="100">
              <template #default="{ row }">{{ formatPercent(row.similarity) }}</template>
            </el-table-column>
          </el-table>
        </template>
      </el-card>

      <el-card shadow="never" class="mt-16">
        <template #header>审稿意见</template>
        <EmptyState v-if="!paper.reviews?.length" description="暂无审稿记录" />
        <el-timeline v-else>
          <el-timeline-item
            v-for="r in paper.reviews"
            :key="r.id"
            :timestamp="formatTime(r.created_at)"
            placement="top"
          >
            <p>
              <strong>{{ r.reviewer?.real_name || r.reviewer?.username || '审稿人' }}</strong>
              <StatusBadge :status="r.status" kind="review" style="margin-left: 8px" />
              <StatusBadge v-if="r.decision" :status="r.decision" kind="decision" style="margin-left: 8px" />
            </p>
            <p class="comment">{{ r.comments || '（暂未提交意见）' }}</p>
          </el-timeline-item>
        </el-timeline>
      </el-card>

      <el-card shadow="never" class="mt-16">
        <template #header>修稿记录</template>
        <EmptyState v-if="!paper.revisions?.length" description="暂无修稿记录" />
        <el-table v-else :data="paper.revisions" size="small" border>
          <el-table-column label="版本" width="80">
            <template #default="{ row }">V{{ row.version }}</template>
          </el-table-column>
          <el-table-column prop="file_name" label="文件" />
          <el-table-column prop="response_letter" label="修改说明" show-overflow-tooltip />
          <el-table-column label="提交人" width="120">
            <template #default="{ row }">{{ row.submitted_by?.real_name || row.submitted_by?.username || '-' }}</template>
          </el-table-column>
          <el-table-column label="时间" width="150">
            <template #default="{ row }">{{ formatTime(row.created_at) }}</template>
          </el-table-column>
        </el-table>
      </el-card>

      <div class="mt-16" style="text-align: right">
        <el-button
          v-if="paper.status === 'revision' && !hasPendingWithdrawal"
          type="primary"
          @click="router.push(`/papers/${paper.id}/revise`)"
        >
          提交修改稿
        </el-button>
        <el-alert
          v-if="hasPendingWithdrawal"
          title="撤稿申请待处理中：审稿、修稿与查重流程已暂停，记录仍可查看"
          type="warning"
          :closable="false"
          class="mt-16"
        />
      </div>
    </template>
  </div>

  <el-dialog v-model="withdrawVisible" title="申请撤稿" width="560px">
    <el-alert
      v-if="needAltNote"
      title="论文处于外审或修稿阶段，须填写替代处理说明"
      type="error"
      :closable="false"
      class="mb"
    />
    <el-form :model="withdrawForm" label-width="110px">
      <el-form-item label="撤稿原因" required>
        <el-input v-model="withdrawForm.reason" type="textarea" :rows="3" placeholder="请说明撤稿原因（至少 5 字）" />
      </el-form-item>
      <el-form-item v-if="needAltNote" label="替代处理说明" required>
        <el-input
          v-model="withdrawForm.altNote"
          type="textarea"
          :rows="3"
          placeholder="外审/修稿中的稿件如何处理，如：已通知审稿人停止审稿、改投其他期刊"
        />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="withdrawVisible = false">取消</el-button>
      <el-button type="danger" :loading="withdrawLoading" @click="submitWithdraw">确认撤稿</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getPaper, getPlagiarism } from '../../api/paper'
import { applyWithdrawal, listPaperWithdrawals } from '../../api/withdrawal'
import type { Paper, PlagiarismResult, WithdrawalItem } from '../../api/types'
import EmptyState from '../../components/EmptyState.vue'
import PaperInfoCard from '../../components/PaperInfoCard.vue'
import PaperStatusSteps from '../../components/PaperStatusSteps.vue'
import StatusBadge from '../../components/StatusBadge.vue'
import { formatPercent, formatTime } from '../../utils/format'

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const paper = ref<Paper | null>(null)
const plagiarism = ref<PlagiarismResult | null>(null)
const withdrawals = ref<WithdrawalItem[]>([])
const withdrawVisible = ref(false)
const withdrawLoading = ref(false)
const withdrawForm = reactive({ reason: '', altNote: '' })

const reportItems = computed(() => {
  if (!plagiarism.value?.report) return []
  try {
    return JSON.parse(plagiarism.value.report) as Array<{ source: string; paragraph: string; similarity: number }>
  } catch {
    return []
  }
})

const hasPendingWithdrawal = computed(() => withdrawals.value.some((w) => w.status === 'pending'))

const canWithdraw = computed(
  () =>
    paper.value &&
    ['submitted', 'initial_review', 'external_review', 'revision'].includes(paper.value.status) &&
    !hasPendingWithdrawal.value
)

const needAltNote = computed(
  () => paper.value && ['external_review', 'revision'].includes(paper.value.status)
)

async function load() {
  const id = route.params.id as string
  loading.value = true
  try {
    paper.value = await getPaper(id)
    plagiarism.value = await getPlagiarism(id)
    withdrawals.value = await listPaperWithdrawals(id)
  } catch {
    // 拦截器已提示
  } finally {
    loading.value = false
  }
}

async function submitWithdraw() {
  if (!paper.value) return
  if (withdrawForm.reason.trim().length < 5) {
    ElMessage.warning('撤稿原因至少 5 字')
    return
  }
  if (needAltNote.value && !withdrawForm.altNote.trim()) {
    ElMessage.warning('外审或修稿中的论文须填写替代处理说明')
    return
  }
  withdrawLoading.value = true
  try {
    await applyWithdrawal(paper.value.id, {
      reason: withdrawForm.reason,
      alt_handling_note: withdrawForm.altNote
    })
    ElMessage.success('撤稿申请已提交，待编辑处理')
    withdrawVisible.value = false
    withdrawForm.reason = ''
    withdrawForm.altNote = ''
    await load()
  } catch {
    // 拦截器已提示
  } finally {
    withdrawLoading.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.mb {
  margin-bottom: 12px;
}
</style>
