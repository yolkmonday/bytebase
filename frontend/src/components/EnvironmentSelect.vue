<template>
  <select
    class="btn-select disabled:cursor-not-allowed"
    :disabled="disabled"
    @change="
      (e) => {
        $emit('select-environment-id', parseInt(e.target.value));
      }
    "
  >
    <option disabled :selected="undefined === state.selectedId">
      {{ $t('environment.select') }}
    </option>
    <template v-for="(environment, index) in environmentList" :key="index">
      <option
        v-if="environment.rowStatus == 'NORMAL'"
        :value="environment.id"
        :selected="environment.id == state.selectedId"
      >
        {{ environmentName(environment) }}
      </option>
      <option
        v-else-if="environment.id == state.selectedId"
        :value="environment.id"
        :selected="true"
      >
        {{ environmentName(environment) }}
      </option>
    </template>
  </select>
</template>

<script lang="ts">
import { computed, reactive, watch } from "vue";
import { useStore } from "vuex";
import cloneDeep from "lodash-es/cloneDeep";
import { Environment } from "../types";

interface LocalState {
  selectedId?: number;
}

export default {
  name: "EnvironmentSelect",
  props: {
    selectedId: {
      type: Number,
    },
    selectDefault: {
      default: true,
      type: Boolean,
    },
    disabled: {
      default: false,
      type: Boolean,
    },
  },
  emits: ["select-environment-id"],
  setup(props, { emit }) {
    const store = useStore();
    const state = reactive<LocalState>({
      selectedId: props.selectedId,
    });

    const environmentList = computed(() => {
      return cloneDeep(
        store.getters["environment/environmentList"](["NORMAL", "ARCHIVED"])
      ).reverse();
    });

    if (environmentList.value && environmentList.value.length > 0) {
      if (
        !props.selectedId ||
        !environmentList.value.find(
          (item: Environment) => item.id == props.selectedId
        )
      ) {
        if (props.selectDefault) {
          for (const environment of environmentList.value) {
            if (environment.rowStatus == "NORMAL") {
              state.selectedId = environment.id;
              emit("select-environment-id", state.selectedId);
              break;
            }
          }
        }
      }
    }

    const invalidateSelectionIfNeeded = () => {
      if (
        state.selectedId &&
        !environmentList.value.find(
          (item: Environment) => item.id == state.selectedId
        )
      ) {
        state.selectedId = undefined;
        emit("select-environment-id", state.selectedId);
      }
    };

    watch(
      () => environmentList.value,
      () => {
        invalidateSelectionIfNeeded();
      }
    );

    watch(
      () => props.selectedId,
      (cur) => {
        state.selectedId = cur;
      }
    );

    return {
      state,
      environmentList,
    };
  },
};
</script>
