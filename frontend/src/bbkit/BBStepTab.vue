<template>
  <div>
    <ol class="flex space-y-0 space-x-8">
      <li v-for="(step, index) in stepItemList" :key="index" class="flex-1">
        <div
          class="group flex flex-col pt-4 border-t-4"
          :class="
            state.currentStep >= index
              ? 'border-accent hover:border-accent-hover cursor-pointer'
              : 'border-control-border'
          "
          @click.prevent="
            () => {
              if (state.currentStep >= index) {
                changeStep(index);
              }
            }
          "
        >
          <div class="flex items-center justify-between">
            <div class="flex flex-col">
              <span
                class="text-xs font-semibold tracking-wide uppercase"
                :class="
                  state.currentStep >= index
                    ? 'text-accent group-hover:text-accent-hover'
                    : 'text-control-light'
                "
                >{{ $t("bbkit.common.step") }} {{ index + 1 }}</span
              >
              <span class="text-sm font-medium">{{ step.title }}</span>
            </div>
            <div
              v-if="state.currentStep > index || state.done"
              class="flex items-center justify-center w-6 h-6 bg-accent text-white rounded-full select-none"
            >
              <heroicons-solid:check class="w-4 h-4" />
            </div>
          </div>
        </div>
      </li>
    </ol>
    <div class="mt-4 mb-4">
      <template v-for="(step, index) in stepItemList" :key="index">
        <slot
          v-if="state.currentStep == index"
          :name="index"
          :next="
            () => {
              changeStep(state.currentStep + 1);
            }
          "
        />
      </template>
    </div>
    <div class="pt-4 border-t border-block-border flex justify-between">
      <button type="button" class="btn-normal" @click.prevent="cancel">
        {{ $t("bbkit.common.cancel") }}
      </button>
      <div class="flex flex-row space-x-2">
        <button
          v-if="state.currentStep != 0"
          type="button"
          class="btn-normal"
          @click.prevent="changeStep(state.currentStep - 1)"
        >
          <heroicons-outline:chevron-left
            class="-ml-1 mr-1 h-5 w-5 text-control-light"
          />
          <span> {{ $t("bbkit.common.back") }}</span>
        </button>
        <button
          v-if="state.currentStep == stepItemList.length - 1"
          :disabled="!allowNext"
          type="button"
          class="btn-primary"
          @click.prevent="finish"
        >
          {{ $t(finishTitle) }}
        </button>
        <button
          v-else-if="!stepItemList[state.currentStep].hideNext"
          :disabled="!allowNext"
          type="button"
          class="btn-primary"
          @click.prevent="changeStep(state.currentStep + 1)"
        >
          {{ $t("bbkit.common.next") }}
        </button>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
import { PropType, reactive } from "vue";
import { BBStepTabItem } from "./types";

interface LocalState {
  done: boolean;
  currentStep: number;
}

export default {
  name: "BBStepTab",
  props: {
    stepItemList: {
      required: true,
      type: Object as PropType<BBStepTabItem[]>,
    },
    allowNext: {
      default: true,
      type: Boolean,
    },
    finishTitle: {
      default: "bbkit.common.finish",
      type: String,
    },
  },
  // For try-change-step and try-finish listener, it needs to call the callback if it determines we can change the step.
  emits: ["try-change-step", "try-finish", "cancel"],
  setup(props, { emit }) {
    const state = reactive<LocalState>({
      done: false,
      currentStep: 0,
    });

    const changeStep = (step: number) => {
      const changeStepCallback = () => {
        state.done = false;
        state.currentStep = step;
      };
      emit("try-change-step", state.currentStep, step, changeStepCallback);
    };

    const finish = () => {
      const finishCallback = () => {
        state.done = true;
      };
      emit("try-finish", finishCallback);
    };

    const cancel = () => {
      emit("cancel");
    };

    return {
      state,
      changeStep,
      finish,
      cancel,
    };
  },
};
</script>
