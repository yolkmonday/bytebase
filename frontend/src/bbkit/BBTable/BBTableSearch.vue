<template>
  <div class="">
    <label for="search" class="sr-only">Search</label>
    <div class="relative rounded-md shadow-sm">
      <div
        class="
          absolute
          inset-y-0
          left-0
          pl-3
          flex
          items-center
          pointer-events-none
        "
        aria-hidden="true"
      >
        <!-- Heroicon name: solid/search -->
        <heroicons-solid:search class="mr-3 h-4 w-4 text-control" />
      </div>
      <input
        id="search"
        type="text"
        autocomplete="off"
        name="search"
        class="
          focus:ring-main focus:border-main
          block
          w-full
          pl-9
          sm:text-sm
          border-control-border
          rounded-md
        "
        :placeholder="placeholder"
        @input="$emit('change-text', $event.target.value)"
      />
    </div>
  </div>
</template>

<script lang="ts">
export default {
  name: "BBTableSearch",
  props: {
    placeholder: {
      default: "Search",
      type: String,
    },
  },
  emits: ["change-text"],
};
</script>
