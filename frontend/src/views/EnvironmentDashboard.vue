<template>
  <div>
    <BBTab
      :tab-item-list="tabItemList"
      :selected-index="state.selectedIndex"
      :reorder-model="state.reorder ? 'ALWAYS' : 'NEVER'"
      @reorder-index="reorderEnvironment"
      @select-index="selectEnvironment"
    >
      <BBTabPanel
        v-for="(item, index) in environmentList"
        :key="item.id"
        :active="index == state.selectedIndex"
      >
        <div v-if="state.reorder" class="flex justify-center pt-5">
          <button
            type="button"
            class="btn-normal py-2 px-4"
            @click.prevent="discardReorder"
          >
            {{ $t('common.cancel') }}
          </button>
          <button
            type="submit"
            class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
            :disabled="!orderChanged"
            @click.prevent="doReorder"
          >
            {{ $t('common.apply') }}
          </button>
        </div>
        <EnvironmentDetail
          v-else
          :environment-slug="environmentSlug(item)"
          @archive="doArchive"
        />
      </BBTabPanel>
    </BBTab>
  </div>
  <BBModal
    v-if="state.showCreateModal"
    :title="$t('environment.create')"
    @close="state.showCreateModal = false"
  >
    <EnvironmentForm
      :create="true"
      :environment="DEFAULT_NEW_ENVIRONMENT"
      :approval-policy="DEFAULT_NEW_APPROVAL_POLICY"
      :backup-policy="DEFAULT_NEW_BACKUP_PLAN_POLICY"
      @create="doCreate"
      @cancel="state.showCreateModal = false"
    />
  </BBModal>

  <BBAlert
    v-if="state.showGuide"
    :style="'INFO'"
    :ok-text="$t('common.do-not-show-again')"
    :cancel-text="$t('common.dismiss')"
    :title="$t('environment.how-to-setup-environment')"
    :description="$t('environment.how-to-setup-environment-description')"
    @ok="
      () => {
        doDismissGuide();
      }
    "
    @cancel="state.showGuide = false"
  >
  </BBAlert>
</template>

<script lang="ts">
import { onMounted, onUnmounted, computed, reactive, watch } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import { array_swap } from "../utils";
import EnvironmentDetail from "../views/EnvironmentDetail.vue";
import EnvironmentForm from "../components/EnvironmentForm.vue";
import { Environment, EnvironmentCreate, Policy, PolicyUpsert } from "../types";
import { BBTabItem } from "../bbkit/types";

const DEFAULT_NEW_ENVIRONMENT: EnvironmentCreate = {
  name: "New Env",
};

// The default value should be consistent with the GetDefaultPolicy from the backend.
const DEFAULT_NEW_APPROVAL_POLICY: PolicyUpsert = {
  payload: {
    value: "MANUAL_APPROVAL_ALWAYS",
  },
};

// The default value should be consistent with the GetDefaultPolicy from the backend.
const DEFAULT_NEW_BACKUP_PLAN_POLICY: PolicyUpsert = {
  payload: {
    schedule: "UNSET",
  },
};

interface LocalState {
  reorderedEnvironmentList: Environment[];
  selectedIndex: number;
  showCreateModal: boolean;
  reorder: boolean;
  showGuide: boolean;
}

export default {
  name: "EnvironmentDashboard",
  components: {
    EnvironmentDetail,
    EnvironmentForm,
  },
  props: {},
  setup() {
    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      reorderedEnvironmentList: [],
      selectedIndex: -1,
      showCreateModal: false,
      reorder: false,
      showGuide: false,
    });

    const selectEnvironmentOnHash = () => {
      if (environmentList.value.length > 0) {
        if (router.currentRoute.value.hash) {
          for (let i = 0; i < environmentList.value.length; i++) {
            if (
              environmentList.value[i].id ==
              router.currentRoute.value.hash.slice(1)
            ) {
              selectEnvironment(i);
              break;
            }
          }
        } else {
          selectEnvironment(0);
        }
      }
    };

    onMounted(() => {
      store.dispatch("command/registerCommand", {
        id: "bb.environment.create",
        registerId: "environment.dashboard",
        run: () => {
          createEnvironment();
        },
      });
      store.dispatch("command/registerCommand", {
        id: "bb.environment.reorder",
        registerId: "environment.dashboard",
        run: () => {
          startReorder();
        },
      });

      selectEnvironmentOnHash();

      if (!store.getters["uistate/introStateByKey"]("guide.environment")) {
        setTimeout(() => {
          state.showGuide = true;
          store.dispatch("uistate/saveIntroStateByKey", {
            key: "environment.visit",
            newState: true,
          });
        }, 1000);
      }
    });

    onUnmounted(() => {
      store.dispatch("command/unregisterCommand", {
        id: "bb.environment.create",
        registerId: "environment.dashboard",
      });
      store.dispatch("command/unregisterCommand", {
        id: "bb.environment.reorder",
        registerId: "environment.dashboard",
      });
    });

    watch(
      () => router.currentRoute.value.hash,
      () => {
        if (router.currentRoute.value.name == "workspace.environment") {
          selectEnvironmentOnHash();
        }
      }
    );

    const environmentList = computed(() => {
      return store.getters["environment/environmentList"]();
    });

    const tabItemList = computed((): BBTabItem[] => {
      if (environmentList.value) {
        const list = state.reorder
          ? state.reorderedEnvironmentList
          : environmentList.value;
        return list.map((item: Environment, index: number) => {
          return {
            title: (index + 1).toString() + ". " + item.name,
            id: item.id,
          };
        });
      }
      return [];
    });

    const createEnvironment = () => {
      stopReorder();
      state.showCreateModal = true;
    };

    const doCreate = (
      newEnvironment: EnvironmentCreate,
      approvalPolicy: Policy,
      backupPolicy: Policy
    ) => {
      store
        .dispatch("environment/createEnvironment", newEnvironment)
        .then((environment: Environment) => {
          Promise.all([
            store.dispatch("policy/upsertPolicyByEnvironmentAndType", {
              environmentId: environment.id,
              type: "bb.policy.pipeline-approval",
              policyUpsert: { payload: approvalPolicy.payload },
            }),
            store.dispatch("policy/upsertPolicyByEnvironmentAndType", {
              environmentId: environment.id,
              type: "bb.policy.backup-plan",
              policyUpsert: { payload: backupPolicy.payload },
            }),
          ]).then(() => {
            state.showCreateModal = false;
            selectEnvironment(environmentList.value.length - 1);
          });
        });
    };

    const doDismissGuide = () => {
      store.dispatch("uistate/saveIntroStateByKey", {
        key: "guide.environment",
        newState: true,
      });
      state.showGuide = false;
    };

    const startReorder = () => {
      state.reorderedEnvironmentList = [...environmentList.value];
      state.reorder = true;
    };

    const stopReorder = () => {
      state.reorder = false;
      state.reorderedEnvironmentList = [];
    };

    const reorderEnvironment = (sourceIndex: number, targetIndex: number) => {
      array_swap(state.reorderedEnvironmentList, sourceIndex, targetIndex);
      selectEnvironment(targetIndex);
    };

    const orderChanged = computed(() => {
      for (let i = 0; i < state.reorderedEnvironmentList.length; i++) {
        if (
          state.reorderedEnvironmentList[i].id != environmentList.value[i].id
        ) {
          return true;
        }
      }
      return false;
    });

    const discardReorder = () => {
      stopReorder();
    };

    const doReorder = () => {
      store
        .dispatch(
          "environment/reorderEnvironmentList",
          state.reorderedEnvironmentList
        )
        .then(() => {
          stopReorder();
        });
    };

    const doArchive = (/* environment: Environment */) => {
      if (environmentList.value.length > 0) {
        selectEnvironment(0);
      }
    };

    const selectEnvironment = (index: number) => {
      state.selectedIndex = index;
      router.replace({
        name: "workspace.environment",
        hash: "#" + environmentList.value[index].id,
      });
    };

    const tabClass = computed(() => "w-1/" + environmentList.value.length);

    return {
      DEFAULT_NEW_ENVIRONMENT,
      DEFAULT_NEW_APPROVAL_POLICY,
      DEFAULT_NEW_BACKUP_PLAN_POLICY,
      state,
      environmentList,
      tabItemList,
      createEnvironment,
      doCreate,
      doArchive,
      doDismissGuide,
      reorderEnvironment,
      orderChanged,
      discardReorder,
      doReorder,
      selectEnvironment,
      tabClass,
    };
  },
};
</script>
