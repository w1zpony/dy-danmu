<script setup lang="ts">
import { ref, computed } from 'vue'
import type { UserGift } from '../types/models/gift_message'
import { ArrowDown, ArrowUp } from '@element-plus/icons-vue'

const props = defineProps<{
  userGifts: UserGift[]
}>()

// 控制展开/折叠状态
const isExpanded = ref(false)

// 计算接收礼物的用户排名
const receiverRankings = computed(() => {
  // 按接收用户ID分组
  const rankMap = new Map<string, { 
    to_user_id: number, 
    to_user_name: string, 
    to_user_display_id: string, 
    totalAmount: number,
    senderCount: number, // 送礼人数
  }>();
  
  // 记录每个接收者的送礼人集合
  const senderSets = new Map<string, Set<number>>();
  
  props.userGifts.forEach(gift => {
    const key = String(gift.to_user_id);
    
    // 初始化接收者数据
    if (!rankMap.has(key)) {
      rankMap.set(key, {
        to_user_id: gift.to_user_id,
        to_user_name: gift.to_user_name,
        to_user_display_id: gift.to_user_display_id,
        totalAmount: 0,
        senderCount: 0,
      });
      senderSets.set(key, new Set());
    }
    
    const userRank = rankMap.get(key)!;
    const senderSet = senderSets.get(key)!;
    
    // 累加总金额
    userRank.totalAmount += gift.total;
    
    // 添加送礼人ID到集合
    senderSet.add(gift.user_id);
    
    // 更新送礼人数
    userRank.senderCount = senderSet.size;
  });
  
  // 转换为数组并按总金额排序
  return Array.from(rankMap.values())
    .sort((a, b) => b.totalAmount - a.totalAmount);
});

// 计算礼物总金额
const totalGiftAmount = computed(() => {
  return props.userGifts.reduce((sum, gift) => sum + gift.total, 0);
});

// 计算总送礼人数（去重）
const totalSenders = computed(() => {
  const uniqueSenders = new Set(props.userGifts.map(gift => gift.user_id));
  return uniqueSenders.size;
});

// 计算总接收人数
const totalReceivers = computed(() => {
  return receiverRankings.value.length;
});

// 切换展开/折叠状态
function toggleExpand() {
  isExpanded.value = !isExpanded.value;
}

// 获取排名标识的样式
function getRankClass(index: number) {
  if (index === 0) return 'bg-yellow-500'; // 金牌
  if (index === 1) return 'bg-gray-400';   // 银牌
  if (index === 2) return 'bg-amber-700';  // 铜牌
  return 'bg-blue-500';                    // 其他
}
</script>

<template>
  <div class="gift-ranking-summary bg-gradient-to-r from-blue-50 to-indigo-50 rounded-xl shadow-md overflow-hidden border border-indigo-100 mb-6">
    <!-- 总金额和展开/折叠按钮 -->
    <div class="flex flex-wrap justify-between items-center p-3 border-b border-indigo-100 bg-white">
      <div class="flex gap-2 sm:gap-4">
        <div>
          <h3 class="text-base sm:text-lg font-medium text-indigo-800">礼物排行</h3>
          <div class="text-xs sm:text-sm text-gray-600">
            总额: <span class="text-orange-500 font-bold">{{ totalGiftAmount }}</span>
          </div>
        </div>
        <div class="px-2 py-1 bg-blue-50 rounded-lg">
          <div class="text-xs text-gray-500">主播</div>
          <div class="text-base sm:text-lg font-medium text-blue-600">{{ totalReceivers }}</div>
        </div>
        <div class="px-2 py-1 bg-green-50 rounded-lg">
          <div class="text-xs text-gray-500">总人数</div>
          <div class="text-base sm:text-lg font-medium text-green-600">{{ totalSenders }}</div>
        </div>
      </div>
      
      <!-- 可点击文本替代按钮 -->
      <div 
        class="flex items-center gap-1 text-indigo-600 cursor-pointer hover:text-indigo-800 transition-colors mt-2 sm:mt-0"
        @click="toggleExpand"
      >
        <span class="text-sm">{{ isExpanded ? '收起' : '展开' }}</span>
        <component :is="isExpanded ? ArrowUp : ArrowDown" class="w-4 h-4" />
      </div>
    </div>
    
    <!-- 完整排行榜 (展开时显示) -->
    <div v-if="isExpanded && receiverRankings.length > 0" class="p-3 bg-white border-t-4 border-indigo-200">
      <div class="overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-200">
          <thead class="bg-gray-50">
            <tr>
              <th scope="col" class="px-2 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider w-12">
                
              </th>
              <th scope="col" class="px-2 py-2 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                主播
              </th>
              <th scope="col" class="px-2 py-2 text-center text-xs font-medium text-gray-500 uppercase tracking-wider w-16">
                人数
              </th>
              <th scope="col" class="px-2 py-2 text-right text-xs font-medium text-gray-500 uppercase tracking-wider w-20">
                总额
              </th>
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-gray-200">
            <tr v-for="(row, index) in receiverRankings" :key="row.to_user_id" class="hover:bg-gray-50">
              <td class="px-2 py-2 whitespace-nowrap">
                <div 
                  :class="[getRankClass(index), 'w-5 h-5 rounded-full flex items-center justify-center text-white font-bold text-xs']"
                >
                  {{ index + 1 }}
                </div>
              </td>
              <td class="px-2 py-2 whitespace-nowrap">
                <div class="flex flex-col">
                  <span class="text-xs font-medium text-gray-800">
                    {{ row.to_user_name }}
                  </span>
                  <span class="text-xs text-gray-500">
                    {{ row.to_user_display_id }}
                  </span>
                </div>
              </td>
              <td class="px-2 py-2 whitespace-nowrap text-center">
                <span class="text-blue-500 font-medium text-xs">{{ row.senderCount }}</span>
              </td>
              <td class="px-2 py-2 whitespace-nowrap text-right">
                <span class="text-orange-500 font-medium text-xs">{{ row.totalAmount }}</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
    
    <!-- 无数据提示 -->
    <div v-if="receiverRankings.length === 0" class="p-3 text-center text-gray-500 bg-white">
      暂无礼物数据
    </div>
  </div>
</template>

<style scoped>
.gift-ranking-summary {
  transition: all 0.3s ease;
}

@media (max-width: 640px) {
  .gift-ranking-summary table {
    font-size: 0.75rem;
  }
  
  .gift-ranking-summary th,
  .gift-ranking-summary td {
    padding: 0.375rem 0.25rem;
  }
}

@media (max-width: 360px) {
  .gift-ranking-summary table {
    font-size: 0.7rem;
  }
}
</style> 