<template>
  <div class="sh-admin-shell">
    <slot name="sidebar" />
    <div class="sh-admin-shell__main">
      <slot name="header" />
      <div class="sh-admin-shell__content">
        <slot />
      </div>
    </div>
  </div>
</template>
