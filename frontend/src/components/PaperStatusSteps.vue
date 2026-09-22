<template>
  <el-steps :active="active" align-center :finish-status="finishStatus" class="paper-steps">
    <el-step title="已提交" description="稿件已提交至期刊" />
    <el-step title="初审中" description="编辑部格式与选题初审" />
    <el-step title="外审中" description="同行专家评审" />
    <el-step title="修改中" description="按意见修改并重投" />
    <el-step :title="endTitle" :description="endDesc" />
  </el-steps>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ status: string }>()

const active = computed(() => {
  switch (props.status) {
    case 'submitted':
      return 1
    case 'initial_review':
      return 2
    case 'external_review':
      return 3
    case 'revision':
      return 4
    case 'accepted':
      return 5
    case 'rejected':
    case 'withdrawn':
      return 4
    default:
      return 0
  }
})

const finishStatus = computed(() => (props.status === 'withdrawn' ? 'wait' : 'success'))

const endTitle = computed(() => {
  if (props.status === 'rejected') return '已拒稿'
  if (props.status === 'withdrawn') return '已撤稿'
  return '已录用'
})
const endDesc = computed(() => {
  if (props.status === 'rejected') return '稿件未通过评审'
  if (props.status === 'withdrawn') return '作者撤稿，流程终止'
  return '稿件被期刊收录'
})
</script>
