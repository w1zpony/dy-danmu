<script setup lang="ts">
import { useNavbar } from '../composables/useNavbar'
import { User } from '@element-plus/icons-vue'

const { handleLogout, isAdmin, authName, navigation } = useNavbar()
</script>

<template>
  <div class="fixed top-0 left-0 right-0 bg-white border-b border-gray-200 h-14 z-50">
    <div class="h-full flex items-center">
      <!-- Logo -->
      <router-link 
        to="/home" 
        class="flex items-center gap-2 hover:opacity-80 transition-all px-4"
      >
        <img 
          src="/NUNU.png" 
          alt="Logo" 
          class="h-7 w-7"
        />
        <div class="flex items-center">
          <span class="text-lg font-bold text-[#409EFF]">Danmu</span>
          <span class="text-lg font-bold text-orange-500">Nu</span>
          <span class="text-xs align-top text-orange-500">+</span>
        </div>
      </router-link>

      <!-- 导航菜单 -->
      <div class="flex items-center">
        <router-link 
          to="/home"
          class="flex items-center px-3 py-2 text-gray-500 hover:text-gray-900 relative border-b-2 border-transparent transition-colors"
          :class="{ 'text-gray-900 border-current nav-active': $route.path === '/home' }"
        >
          <span class="font-medium">首页</span>
        </router-link>
        
        <router-link 
          to="/users"
          v-if="isAdmin"
          class="flex items-center px-3 py-2 text-gray-500 hover:text-gray-900 relative border-b-2 border-transparent transition-colors"
          :class="{ 'text-gray-900 border-current nav-active': $route.path.includes('users') }"
        >
          <span class="font-medium">用户管理</span>
        </router-link>
        
        <router-link 
          v-for="item in navigation.filter(n => !['home', 'users'].includes(n.path.slice(1)))"
          :key="item.path"
          :to="item.path"
          v-show="!item.requireAdmin || isAdmin"
          class="flex items-center px-3 py-2 text-gray-500 hover:text-gray-900 relative border-b-2 border-transparent transition-colors"
          :class="{ 'text-gray-900 border-current nav-active': $route.path.includes(item.path.slice(1)) }"
        >
          <span class="font-medium">{{ item.name }}</span>
        </router-link>
      </div>

      <!-- 用户菜单 - 靠右对齐 -->
      <div class="ml-auto pr-4">
        <el-dropdown trigger="click">
          <div class="flex items-center px-3 py-1.5 text-gray-700 cursor-pointer transition-all border rounded-md hover:bg-gray-50">
            <el-icon class="mr-2"><User /></el-icon>
            <span class="font-medium max-w-[120px] truncate">{{ authName }}</span>
          </div>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item>
                <router-link to="/profile" class="text-gray-700 font-medium">个人资料</router-link>
              </el-dropdown-item>
              <el-dropdown-item v-if="isAdmin">
                <router-link to="/users" class="text-gray-700 font-medium">用户管理</router-link>
              </el-dropdown-item>
              <el-dropdown-item divided>
                <span class="text-red-600 font-medium block w-full" @click="handleLogout">登出</span>
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
  border-color: #f97316;  /* Tailwind's orange-500 color */
}

.router-link-active {
  color: rgb(17 24 39);  /* text-gray-900 */
  border-color: #f97316;
}
</style> 