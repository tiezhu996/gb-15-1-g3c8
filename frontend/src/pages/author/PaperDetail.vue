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

      <div v-if="paper.withdrawal" class="mt-16">
        <WithdrawalPanel :withdrawal="paper.withdrawal" border />
      </div>

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
        <el-button v-if="canRevise" type="primary" @click="router.push(`/papers/${paper.id}/revise`)">
          提交修改稿
        </el-button>
        <el-button v-if="canWithdraw" type="danger" plain @click="withdrawVisible = true">
          申请撤稿
        </el-button>
      </div>
    </template>
  </div>

  <el-dialog v-model="withdrawVisible" title="申请撤稿" width="560px">
    <el-alert
      type="info"
      :closable="false"
      :title="`论文当前状态：${paperStatusText(paper?.status || '')}`"
      description="撤稿申请提交后须等待编辑部处理，期间流程暂停；批准后论文进入已撤稿终态，已录用论文不可撤稿。"
      class="mb"
    />
    <el-form ref="withdrawFormRef" :model="withdrawForm" :rules="withdrawRules" label-width="110px">
      <el-form-item label="撤稿原因" prop="reason">
        <el-input v-model="withdrawForm.reason" type="textarea" :rows="3" placeholder="请填写撤稿原因（至少 10 字）" />
      </el-form-item>
      <el-form-item v-if="alternativeRequired" label="替代处理说明" prop="alternative_note">
        <el-input
          v-model="withdrawForm.alternative_note"
          type="textarea"
          :rows="3"
          placeholder="外审/修稿阶段必须填写替代处理说明（至少 10 字）"
        />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="withdrawVisible = false">取消</el-button>
      <el-button type="danger" :loading="withdrawLoading" @click="submitWithdraw">确认申请撤稿</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import type { FormInstance } from 'element-plus'
import { applyWithdrawal, getPaper, getPlagiarism } from '../../api/paper'
import type { Paper, PlagiarismResult } from '../../api/types'
import EmptyState from '../../components/EmptyState.vue'
import PaperInfoCard from '../../components/PaperInfoCard.vue'
import PaperStatusSteps from '../../components/PaperStatusSteps.vue'
import StatusBadge from '../../components/StatusBadge.vue'
import WithdrawalPanel from '../../components/WithdrawalPanel.vue'
import { formatPercent, formatTime, paperStatusText } from '../../utils/format'

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const paper = ref<Paper | null>(null)
const plagiarism = ref<PlagiarismResult | null>(null)

const withdrawVisible = ref(false)
const withdrawLoading = ref(false)
const withdrawFormRef = ref<FormInstance>()
const withdrawForm = reactive({ reason: '', alternative_note: '' })

const pendingWithdrawal = computed(() => paper.value?.withdrawal?.status === 'pending')
const canRevise = computed(() => paper.value?.status === 'revision' && !pendingWithdrawal.value)
const canWithdraw = computed(() =>
  !!paper.value &&
  ['submitted', 'initial_review', 'external_review', 'revision'].includes(paper.value.status) &&
  !pendingWithdrawal.value
)
const alternativeRequired = computed(() =>
  ['external_review', 'revision'].includes(paper.value?.status || '')
)
const withdrawRules = computed(() => ({
  reason: [{ required: true, min: 10, message: '撤稿原因至少 10 字', trigger: 'blur' }],
  alternative_note: alternativeRequired.value
    ? [{ required: true, min: 10, message: '外审/修稿阶段必须填写替代处理说明（至少 10 字）', trigger: 'blur' }]
    : []
}))

const reportItems = computed(() => {
  if (!plagiarism.value?.report) return []
  try {
    return JSON.parse(plagiarism.value.report) as Array<{ source: string; paragraph: string; similarity: number }>
  } catch {
    return []
  }
})

async function load() {
  const id = route.params.id as string
  loading.value = true
  try {
    paper.value = await getPaper(id)
    plagiarism.value = await getPlagiarism(id)
  } catch {
    // 拦截器已提示
  } finally {
    loading.value = false
  }
}

async function submitWithdraw() {
  if (!paper.value) return
  const valid = await withdrawFormRef.value?.validate().catch(() => false)
  if (!valid) return
  withdrawLoading.value = true
  try {
    await applyWithdrawal(paper.value.id, {
      reason: withdrawForm.reason,
      alternative_note: alternativeRequired.value ? withdrawForm.alternative_note : undefined
    })
    ElMessage.success('撤稿申请已提交，等待编辑部处理')
    withdrawVisible.value = false
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
  margin-bottom: 4px;
}
</style>
