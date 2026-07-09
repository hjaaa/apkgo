# Vue 组件规约

## 前言

本规约适用于 Vue 3 单文件组件（SFC），统一使用 **Composition API + `<script setup lang="ts">`**，不使用 Options API。

条目中标注的 eslint 规则来自 `eslint-plugin-vue`，是否已启用以所在工程的 lint 配置为准；未被 lint 覆盖的条目须人工遵循。

## 1 组件命名

- 1.1 【强制】组件名必须由多个单词组成。eslint: [vue/multi-word-component-names](https://eslint.vuejs.org/rules/multi-word-component-names.html)

  避免与现有及未来的 HTML 元素冲突（HTML 元素都是单个单词）。

  **例外**：根组件 `App.vue`，以及按目录约定命名的路由页面组件（如 `views/<模块>/index.vue`）；豁免范围应在工程 eslint 配置中显式声明。

  ```vue
  <!-- bad -->
  <Item />

  <!-- good -->
  <TodoItem />
  ```

- 1.2 【推荐】通用组件文件名使用 PascalCase；路由页面组件固定为 `views/<模块>/index.vue`。

  PascalCase 与编辑器自动补全、JS 中的引用方式一致。JS/TS 中导入和使用组件同样使用 PascalCase。

  ```
  # bad
  components/
  ├── mycomponent.vue
  ├── myComponent.vue

  # good
  components/
  ├── MyComponent.vue
  ```

- 1.3 【推荐】展示应用级样式和约定的基础组件，统一使用 `Base` 前缀。

  字母排序时自然聚拢，便于识别与统一维护。

  ```
  # bad
  components/
  ├── MyButton.vue
  ├── Icon.vue

  # good
  components/
  ├── BaseButton.vue
  ├── BaseIcon.vue
  ```

- 1.4 【推荐】只在某个父组件场景下使用的紧耦合子组件，以父组件名作为前缀。

  体现组件关系，且字母排序时相关文件相邻，避免深层目录嵌套。

  ```
  # bad
  components/
  ├── TodoList.vue
  ├── TodoItem.vue
  ├── TodoButton.vue

  # good
  components/
  ├── TodoList.vue
  ├── TodoListItem.vue
  ├── TodoListItemButton.vue
  ```

- 1.5 【推荐】组件名从最高层级（最一般）的词开始，以描述性修饰词结尾。

  ```
  # bad
  ├── ClearSearchButton.vue
  ├── RunSearchButton.vue

  # good
  ├── SearchButtonClear.vue
  ├── SearchButtonRun.vue
  ```

- 1.6 【推荐】组件名使用完整单词，不使用缩写。

  编辑器补全让长名字的输入成本很低，而完整单词带来的明确性价值很高。

  ```
  # bad
  ├── UProfOpts.vue

  # good
  ├── UserProfileOptions.vue
  ```

## 2 Props 与组件通信

- 2.1 【强制】prop 定义必须尽可能详细，至少指定类型。

  TypeScript 工程优先使用**类型声明式** `defineProps`，必填/可选由类型表达，默认值用 `withDefaults`；需要运行时校验时再改用运行时声明并提供 `validator`。

  ```vue
  <script setup lang="ts">
  // bad
  const props = defineProps(['status'])

  // good - 类型声明式（首选）
  const props = withDefaults(
    defineProps<{
      status: 'syncing' | 'synced' | 'error'
      count?: number
    }>(),
    { count: 0 },
  )
  </script>
  ```

- 2.2 【强制】父子组件通信遵循"prop 向下、事件向上"，禁止在子组件中修改 prop，禁止访问 `$parent`。eslint: [vue/no-mutating-props](https://eslint.vuejs.org/rules/no-mutating-props.html)

  隐式的父子通信（改 prop、摸 `$parent`）造成紧耦合，父组件状态变化难以追踪。

  ```vue
  <!-- bad：v-model 直接绑定 prop 对象的属性，等于子组件改父组件状态 -->
  <script setup lang="ts">
  defineProps<{ todo: Todo }>()
  </script>
  <template>
    <input v-model="todo.text" />
  </template>

  <!-- good：值进事件出 -->
  <script setup lang="ts">
  defineProps<{ todo: Todo }>()
  const emit = defineEmits<{ input: [value: string] }>()
  </script>
  <template>
    <input :value="todo.text" @input="emit('input', ($event.target as HTMLInputElement).value)" />
  </template>
  ```

- 2.3 【推荐】组件对外发出的事件用 `defineEmits` 显式声明，TypeScript 工程使用类型声明式写法（见 2.2 示例）。

## 3 模板

- 3.1 【强制】`v-for` 必须绑定 `key`。eslint: [vue/require-v-for-key](https://eslint.vuejs.org/rules/require-v-for-key.html)

  key 用于维护列表项的组件状态与对象恒定性，保证 diff 与动画行为可预测。key 使用业务主键，不要用数组下标。

  ```vue
  <!-- bad -->
  <li v-for="todo in todos">{{ todo.text }}</li>

  <!-- good -->
  <li v-for="todo in todos" :key="todo.id">{{ todo.text }}</li>
  ```

- 3.2 【强制】禁止在同一元素上同时使用 `v-if` 和 `v-for`。eslint: [vue/no-use-v-if-with-v-for](https://eslint.vuejs.org/rules/no-use-v-if-with-v-for.html)

  过滤列表用 computed；条件渲染个别项用 `<template>` 包一层 `v-for`。

  ```vue
  <!-- bad -->
  <li v-for="user in users" v-if="user.isActive" :key="user.id">{{ user.name }}</li>

  <!-- good：computed 过滤 -->
  <li v-for="user in activeUsers" :key="user.id">{{ user.name }}</li>

  <!-- good：template 包装 -->
  <template v-for="user in users" :key="user.id">
    <li v-if="user.isActive">{{ user.name }}</li>
  </template>
  ```

- 3.3 【推荐】模板中组件使用 PascalCase 标签，无内容的组件自闭合。

  PascalCase 与自定义 HTML 元素在视觉上区分明显，编辑器可自动补全；自闭合表明组件没有（也不应有）插槽内容。

  ```vue
  <!-- bad -->
  <my-component></my-component>

  <!-- good -->
  <MyComponent />
  ```

- 3.4 【推荐】prop 声明用 camelCase；模板中传参用 kebab-case。

  声明遵循 JavaScript 命名约定，模板传参与 HTML 属性风格一致。同一写法在同一工程内保持统一。

  ```vue
  <!-- bad -->
  <WelcomeMessage greetingText="hi" />

  <!-- good -->
  <WelcomeMessage greeting-text="hi" />
  ```

- 3.5 【推荐】模板中只写简单表达式；复杂逻辑提取为 computed 或函数。

  模板表达的是"显示什么"，而不是"怎么计算"。

  ```vue
  <!-- bad -->
  {{ fullName.split(' ').map((w) => w[0].toUpperCase() + w.slice(1)).join(' ') }}

  <!-- good -->
  {{ normalizedFullName }}
  ```

- 3.6 【推荐】复杂的 computed 拆分为多个简单 computed。

  更易测试、易读、易适应需求变化。

  ```ts
  // bad
  const price = computed(() => {
    const basePrice = manufactureCost.value / (1 - profitMargin.value)
    return basePrice - basePrice * (discountPercent.value || 0)
  })

  // good
  const basePrice = computed(() => manufactureCost.value / (1 - profitMargin.value))
  const discount = computed(() => basePrice.value * (discountPercent.value || 0))
  const finalPrice = computed(() => basePrice.value - discount.value)
  ```

- 3.7 【推荐】非空属性值始终加引号；多属性元素的换行交给 Prettier 处理，以格式化结果为准。

  ```vue
  <!-- bad -->
  <input type=text>

  <!-- good -->
  <input type="text" />
  ```

- 3.8 【推荐】统一使用指令缩写：`:` 代替 `v-bind:`，`@` 代替 `v-on:`，`#` 代替 `v-slot:`，同一工程内不混用全称。

  ```vue
  <!-- bad：缩写与全称混用 -->
  <input v-bind:value="text" @input="onInput" />

  <!-- good -->
  <input :value="text" @input="onInput" />
  ```

- 3.9 【参考】元素属性按统一顺序书写：`is` → `v-for` → `v-if`/`v-else-if`/`v-else`/`v-show` → `v-pre`/`v-once` → `id` → `ref`/`key` → `v-model` → 其他属性 → `@` 事件 → `v-html`/`v-text`。eslint: [vue/attributes-order](https://eslint.vuejs.org/rules/attributes-order.html)

## 4 SFC 结构与样式

- 4.1 【强制】SFC 顶级块顺序统一为 `<template>` → `<script setup>` → `<style>`。

  统一顺序便于快速定位各块，`<style>` 固定放最后。

  ```vue
  <template>
    <!-- ... -->
  </template>

  <script setup lang="ts">
  // ...
  </script>

  <style scoped>
  /* ... */
  </style>
  ```

- 4.2 【强制】组件样式必须作用域化（`<style scoped>`），根组件 `App.vue` 与全局布局组件除外。

  防止样式泄漏到其他组件或被第三方 CSS 干扰。使用原子化 CSS 框架（如 Tailwind）的工程优先使用工具类；确需自定义样式时写入 `<style scoped>`。

- 4.3 【推荐】`scoped` 样式中使用类选择器，不使用元素选择器。

  scoped 通过 `data-v-xxx` 属性实现，`button[data-v-xxx]` 这类"元素+属性"选择器的匹配性能显著低于"类+属性"选择器。

  ```vue
  <!-- bad -->
  <style scoped>
  button {
    background-color: red;
  }
  </style>

  <!-- good -->
  <style scoped>
  .button-close {
    background-color: red;
  }
  </style>
  ```

- 4.4 【参考】多行的 prop 定义之间、逻辑块之间可用空行分组，提升长 `<script setup>` 的可扫描性。

## 5 状态管理

- 5.1 【推荐】跨组件共享的状态统一放 Pinia store；不要用事件总线、层层透传 prop 或可变全局对象模拟全局状态。

  组件内部状态用 `ref`/`reactive` 即可，不要为只有单个组件使用的状态建 store。
