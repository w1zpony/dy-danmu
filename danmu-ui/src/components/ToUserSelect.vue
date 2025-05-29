<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { getToUsers } from '../api/gift_message'
import type { User } from '../types/models/user'
import { Refresh } from '@element-plus/icons-vue'

const props = defineProps<{
  modelValue: number[]  // 用于 v-model 绑定
  roomDisplayId: string // 房间ID
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: number[]): void
}>()

const userList = ref<User[]>([])
const selectedUserIds = ref<number[]>(props.modelValue || [])

function resetAndFetch() {
  // 重置所有状态到初始值
  selectedUserIds.value = []
  emit('update:modelValue', [])  // 通知父组件清空选择
  
  // 重新获取数据
  fetchUsers()
}

async function fetchUsers() {
  try {
    const res = await getToUsers(props.roomDisplayId)
    userList.value = [
      {
        id: 0,
        user_id: 0,
        display_id: '主播',
        user_name: '主播'
      },
      ...res.data.data
    ]
    
    // 在数据加载后应用滚动样式
    nextTick(() => {
      applyScrollStyles()
    })
  } catch (error) {
    console.error('[ToUsers] Fetch error:', error)
  }
}

function handleSelectChange(values: number[]) {
  selectedUserIds.value = values
  emit('update:modelValue', values)  // 添加这行，确保更新父组件
}

// 手动应用滚动样式
function applyScrollStyles() {
  const dropdowns = document.querySelectorAll('.user-select-dropdown .el-scrollbar__wrap')
  dropdowns.forEach(dropdown => {
    if (dropdown instanceof HTMLElement) {
      dropdown.style.maxHeight = '300px'
      dropdown.style.overflowY = 'auto'
    }
  })
}

onMounted(() => {
  fetchUsers()
  
  // 监听下拉菜单打开事件
  document.addEventListener('click', (e) => {
    // 延迟一点时间确保下拉菜单已经渲染
    setTimeout(applyScrollStyles, 100)
  })
})
</script>

<template>
  <div>
    <el-select
      v-model="selectedUserIds"
      multiple
      collapse-tags
      collapse-tags-tooltip
      :style="{ width: '240px' }"
      :popper-class="'width-240px user-select-dropdown'"
      placeholder="请选接收用户"
      @change="handleSelectChange"
    >
      <!-- 刷新按钮移到下拉菜单顶部 -->
      <template #prefix>
        <el-button
          type="primary"
          link
          :icon="Refresh"
          @click.stop="resetAndFetch"
        />
      </template>

      <el-option
        v-for="item in userList"
        :key="item.user_id"
        :label="`${item.user_name}`"
        :value="item.user_id"
      >
        <div class="flex items-center space-x-1 px-2 py-1 w-full">
          <el-tag 
            type="info" 
            size="large"
            class="!w-[120px] !truncate !text-center !m-0 shrink-0"
          >
            {{ item.display_id }}
          </el-tag>
          <el-tag 
            type="success" 
            size="large"
            class="!w-[140px] !truncate !text-center !m-0 shrink-0"
          >
            {{ item.user_name }}
          </el-tag>
        </div>
      </el-option>
    </el-select>
  </div>
</template>

<style>
.width-240px {
  min-width: 280px;  /* 增加宽度以适应两个 tag */
}

.user-select-dropdown .el-select-dropdown__list {
  padding: 0;
}

.user-select-dropdown .el-select-dropdown__item {
  padding: 0;
}
</style>