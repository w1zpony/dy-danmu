<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { getGiftRanking } from '../api/gift_message'
import type { UserGift } from '../types/models/gift_message'
import TimeRangePicker from './TimeRangePicker.vue'
import ToUserSelect from './ToUserSelect.vue'
import GiftReceiverRanking from './GiftReceiverRanking.vue'
import dayjs from 'dayjs'

const props = defineProps<{
  roomDisplayId: string
}>()

// 初始化为今日的开始和结束时间
const timeRange = ref<[number, number]>([
  dayjs().startOf('day').valueOf(),
  dayjs().endOf('day').valueOf()
])
const selectedToUserIds = ref<number[]>([])
const giftList = ref<UserGift[]>([])
const loading = ref(false)

// 分页相关的响应式变量
const currentPage = ref(1)
const pageSize = ref(50)
const total = ref(0)

// 当前页数据
const currentPageData = ref<UserGift[]>([])

const tableContainer = ref<HTMLElement | null>(null)

// 计算分页数据
function updateCurrentPageData() {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  currentPageData.value = giftList.value.slice(start, end)
}

// 处理页码改变（不滚动）
function handleTopPageChange(page: number) {
  currentPage.value = page
  updateCurrentPageData()
}

// 处理底部分页器页码改变（滚动到表格顶部）
function handleBottomPageChange(page: number) {
  currentPage.value = page
  updateCurrentPageData()
  nextTick(() => {
    tableContainer.value?.scrollIntoView({behavior: 'smooth'})
  })
}

// 处理每页条数改变
function handleSizeChange(size: number) {
  pageSize.value = size
  currentPage.value = 1
  updateCurrentPageData()
}

// 展开行的懒加载控制
const expandedRows = ref(new Set<string>())

// 处理展开行事件
function handleExpand(row: UserGift, expanded: boolean) {
  const rowKey = `${row.user_display_id}-${row.to_user_display_id}`
  if (expanded) {
    expandedRows.value.add(rowKey)
  } else {
    expandedRows.value.delete(rowKey)
  }
}

// 判断是否需要渲染展开行内容
function shouldRenderExpanded(row: UserGift) {
  const rowKey = `${row.user_display_id}-${row.to_user_display_id}`
  return expandedRows.value.has(rowKey)
}

// 获取数据
async function fetchGiftRanking() {
  loading.value = true
  currentPage.value = 1  // 重置到第一页
  try {
    const res = await getGiftRanking({
      room_display_id: props.roomDisplayId,
      to_user_ids: selectedToUserIds.value.map(Number),
      begin: timeRange.value[0],
      end: timeRange.value[1]
    })
    giftList.value = res.data.data
    total.value = giftList.value.length
    updateCurrentPageData()
  } catch (error) {
    console.error('[GiftRanking] Fetch error:', error)
  } finally {
    loading.value = false
  }
}

// 组件挂载时自动查询
onMounted(fetchGiftRanking)

function formatTime(timestamp: number) {
  return dayjs(timestamp).format('YYYY-MM-DD HH:mm:ss')
}

// 获取排序后的礼物列表
function getSortedGiftList(gifts: UserGift['gift_list']) {
  return [...gifts].sort((a, b) => b.timestamp - a.timestamp)
}

// 计算礼物总金额
</script>

<template>
  <div class="space-y-4">
    <!-- 筛选条件和查询按钮 -->
    <div class="flex flex-wrap items-end gap-4">
      <TimeRangePicker v-model="timeRange" />
      <ToUserSelect 
        v-model="selectedToUserIds"
        :room-display-id="roomDisplayId"
      />
      <el-button 
        type="primary"
        :loading="loading"
        @click="fetchGiftRanking"
      >
        查询
      </el-button>
    </div>

    <!-- 礼物接收排行榜 -->
    <GiftReceiverRanking :user-gifts="giftList" />

    <!-- 礼物排行列表 -->
    <div
        ref="tableContainer"
        class="overflow-x-auto"
    >
      <!-- 顶部分页器 -->
      <div v-if="total > 0" class="flex justify-end mb-4">
        <el-pagination
            v-model:current-page="currentPage"
            v-model:page-size="pageSize"
            :page-sizes="[20, 50, 100]"
            :total="total"
            :pager-count="5"
            size="small"
            layout="total, sizes, prev, pager, next"
            @size-change="handleSizeChange"
            @current-change="handleTopPageChange"
        />
      </div>

      <el-table
          :data="currentPageData"
          class="w-full"
          @expand-change="handleExpand"
      >
        <el-table-column type="expand" width="20">
          <template #default="{ row }">
            <div v-if="shouldRenderExpanded(row)" class="p-4 bg-gray-50">
              <el-table :data="getSortedGiftList(row.gift_list)" class="w-full">
                <el-table-column label="礼物信息" min-width="200">
                  <template #default="{ row: gift }">
                    <div class="flex items-center gap-2">
                      <el-image
                          :src="gift.image"
                          class="w-8 h-8 object-cover rounded"
                          fit="cover"
                      />
                      <span>{{ gift.message }}</span>
                    </div>
                  </template>
                </el-table-column>

                <el-table-column label="价值" min-width="100" align="right">
                  <template #default="{ row: gift }">
                    <div class="flex flex-col items-end">
                      <span class="text-orange-500">
                        {{ gift.diamond_count * gift.combo_count }}
                      </span>
                      <span class="text-gray-400 text-xs">
                        {{ gift.diamond_count }}×{{ gift.combo_count }}
                      </span>
                    </div>
                  </template>
                </el-table-column>

                <el-table-column label="时间" min-width="160">
                  <template #default="{ row: gift }">
                    {{ formatTime(gift.timestamp) }}
                  </template>
                </el-table-column>
              </el-table>
            </div>
          </template>
        </el-table-column>

        <el-table-column width="22" align="center">
          <template #default="{ $index }">
            <template v-if="(currentPage - 1) * pageSize + $index + 1 <= 50">
              <span
                  class="text-[11px] font-medium inline-block leading-none"
                  :class="{
                  'text-yellow-500': (currentPage - 1) * pageSize + $index === 0,
                  'text-gray-500': (currentPage - 1) * pageSize + $index === 1,
                  'text-orange-500': (currentPage - 1) * pageSize + $index === 2,
                  'text-blue-500': (currentPage - 1) * pageSize + $index > 2
                }"
              >
                {{ (currentPage - 1) * pageSize + $index + 1 }}
              </span>
            </template>
          </template>
        </el-table-column>

        <el-table-column label="送礼用户" min-width="90">
          <template #default="{ row }">
            <div class="flex flex-col">
              <span class="text-[11px] text-green-600 break-all">
                {{ row.user_name }}
              </span>
              <el-tag size="small" type="info" class="!text-xs">
                {{ row.user_display_id }}
              </el-tag>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="接收用户" min-width="90">
          <template #default="{ row }">
            <div class="flex flex-col">
              <span class="text-[11px] text-blue-600 break-all">
                {{ row.to_user_name }}
              </span>
              <el-tag
                  v-if="row.to_user_display_id"
                  size="small"
                  type="info"
                  class="!text-xs"
              >
                {{ row.to_user_display_id }}
              </el-tag>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="金额" min-width="68" align="right">
          <template #default="{ row }">
            <span class="text-orange-500 font-medium text-[11px] tabular-nums">
              {{ row.total }}
            </span>
          </template>
        </el-table-column>
      </el-table>

      <!-- 底部分页器 -->
      <div class="flex justify-end mt-4">
        <el-pagination
            v-model:current-page="currentPage"
            v-model:page-size="pageSize"
            :page-sizes="[20, 50, 100]"
            :total="total"
            :pager-count="5"
            size="small"
            layout="total, sizes, prev, pager, next"
            @size-change="handleSizeChange"
            @current-change="handleBottomPageChange"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.el-table :deep(.el-table__expanded-cell) {
  padding: 0;
}

@media (max-width: 640px) {
  .el-table {
    font-size: 14px;
  }
}

.el-table tr {
  cursor: pointer;
}

.el-table tr:hover > td {
  background-color: var(--el-table-row-hover-bg-color);
}
</style>