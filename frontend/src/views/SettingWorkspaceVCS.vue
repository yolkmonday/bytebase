<template>
  <div class="space-y-4">
    <div class="textinfolabel">
      {{ $t('version-control.setting.description') }}
    </div>
    <div class="flex items-center justify-end">
      <button
        type="button"
        class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
        @click.prevent="addVCSProvider"
      >
        {{ $t('version-control.setting.add-git-provider.self') }}
      </button>
    </div>
    <div class="pt-4 border-t">
      <div v-if="vcsList.length > 0" class="space-y-6">
        <template v-for="(vcs, index) in vcsList" :key="index">
          <VCSCard :vcs="vcs" />
        </template>
      </div>
      <template v-else>
        <VCSSetupWizard />
      </template>
    </div>
  </div>
</template>

<script lang="ts">
import { reactive, computed, watchEffect } from "vue";
import { useRouter } from "vue-router";
import VCSCard from "../components/VCSCard.vue";
import VCSSetupWizard from "../components/VCSSetupWizard.vue";
import { useStore } from "vuex";

// eslint-disable-next-line @typescript-eslint/no-empty-interface
interface LocalState {}

export default {
  name: "SettingWorkspaceVCS",
  components: {
    VCSCard,
    VCSSetupWizard,
  },
  setup() {
    const store = useStore();
    const router = useRouter();
    const state = reactive<LocalState>({});

    const prepareVCSList = () => {
      store.dispatch("vcs/fetchVCSList");
    };

    watchEffect(prepareVCSList);

    const vcsList = computed(() => {
      return store.getters["vcs/vcsList"]();
    });

    const addVCSProvider = () => {
      router.push({
        name: "setting.workspace.version-control.create",
      });
    };

    return {
      state,
      vcsList,
      addVCSProvider,
    };
  },
};
</script>
