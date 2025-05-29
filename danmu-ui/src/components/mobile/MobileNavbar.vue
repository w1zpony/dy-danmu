<script setup lang="ts">
import { useNavbar } from '../../composables/useNavbar'
import { User } from '@element-plus/icons-vue'

const { handleLogout, isAdmin, authName, navigation } = useNavbar()
</script>

<template>
  <div class="fixed top-0 left-0 right-0 bg-white border-b border-gray-200 h-12 z-50">
    <div class="h-full flex items-center">
      <!-- Logo -->
      <router-link 
        to="/home" 
        class="flex items-center gap-2 hover:opacity-80 transition-all px-3"
      >

        <div class="flex items-center">
          <span class="text-base font-bold text-[#409EFF]">Danmu</span>
          <span class="text-base font-bold text-orange-500">Nu</span>
          <span class="text-xs align-top text-orange-500">+</span>
        </div>
      </router-link>

      <!-- 导航菜单 -->
      <div class="flex items-center">
        <router-link 
          to="/home"
          class="flex items-center px-2 py-1.5 text-gray-500 hover:text-gray-900 relative border-b-2 border-transparent transition-colors text-sm"
          :class="{ 'text-gray-900 border-current nav-active': $route.path === '/home' }"
        >
          <span class="font-medium">首页</span>
        </router-link>
        
        <router-link 
          to="/users"
          v-if="isAdmin"
          class="flex items-center px-2 py-1.5 text-gray-500 hover:text-gray-900 relative border-b-2 border-transparent transition-colors text-sm"
          :class="{ 'text-gray-900 border-current nav-active': $route.path.includes('users') }"
        >
          <span class="font-medium">用户管理</span>
        </router-link>
        
        <router-link 
          v-for="item in navigation.filter(n => !['home', 'users'].includes(n.path.slice(1)))"
          :key="item.path"
          :to="item.path"
          v-show="!item.requireAdmin || isAdmin"
          class="flex items-center px-2 py-1.5 text-gray-500 hover:text-gray-900 relative border-b-2 border-transparent transition-colors text-sm"
          :class="{ 'text-gray-900 border-current nav-active': $route.path.includes(item.path.slice(1)) }"
        >
          <span class="font-medium">{{ item.name }}</span>
        </router-link>
      </div>

      <!-- 用户菜单 -->
      <div class="ml-auto pr-3">
        <el-dropdown trigger="click" placement="bottom-end">
          <div class="flex items-center px-2 py-1 text-gray-700 cursor-pointer transition-all border rounded-md hover:bg-gray-50 text-sm">
            <el-icon class="mr-1"><User /></el-icon>
            <span class="font-medium max-w-[80px] truncate">{{ authName }}</span>
          </div>
          <template #dropdown>
            <el-dropdown-menu class="!min-w-[120px]">
              <el-dropdown-item>
                <router-link to="/profile" class="flex items-center justify-center text-gray-700">
                  个人资料
                </router-link>
              </el-dropdown-item>
              <el-dropdown-item v-if="isAdmin">
                <router-link to="/users" class="flex items-center justify-center text-gray-700">
                  用户管理
                </router-link>
              </el-dropdown-item>
              <el-dropdown-item divided>
                <div 
                  class="flex items-center justify-center text-red-600 w-full" 
                  @click="handleLogout"
                >
                  登出
                </div>
              </el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </div>
  </div>
</template>

<style scoped>
.nav-active {
  border-color: #f97316;
}

.router-link-active {
  color: rgb(17 24 39);
  border-color: #f97316;
}

:deep(.el-dropdown-menu) {
  --el-dropdown-menuItem-hover-fill: rgb(249 250 251);
  --el-dropdown-menuItem-hover-color: currentColor;
}

:deep(.el-dropdown-menu__item) {
  padding: 8px 12px;
  justify-content: center;
}
</style>
