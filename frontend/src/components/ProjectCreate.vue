<template>
  <form class="px-4 py-2 space-y-6 divide-y divide-block-border">
    <div class="grid gap-y-6 gap-x-4 grid-cols-1">
      <div class="col-span-1">
        <label for="name" class="text-lg leading-6 font-medium text-control">
          {{ $t("project.create-modal.project-name")
          }}<span class="text-red-600">*</span>
        </label>
        <BBTextField
          class="mt-4 w-full"
          :required="true"
          :placeholder="'Project name'"
          :value="state.project.name"
          @input="state.project.name = $event.target.value"
        />
      </div>
      <div class="col-span-1">
        <label for="name" class="text-lg leading-6 font-medium text-control">
          {{ $t("project.create-modal.key") }}
          <span class="text-red-600">*</span>
          <span class="text-sm font-normal">
            {{ $t("project.create-modal.key-hint") }}</span
          >
        </label>
        <BBTextField
          class="mt-4 w-full uppercase"
          :required="true"
          :value="state.project.key"
          @input="state.project.key = $event.target.value"
        />
      </div>
    </div>
    <!-- Create button group -->
    <div class="pt-4 flex justify-end">
      <button
        type="button"
        class="btn-normal py-2 px-4"
        @click.prevent="cancel"
      >
        {{ $t("common.cancel") }}
      </button>
      <button
        class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
        :disabled="!allowCreate"
        @click.prevent="create"
      >
        {{ $t("common.create") }}
      </button>
    </div>
  </form>
</template>

<script lang="ts">
import { computed, reactive, onMounted, onUnmounted } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import isEmpty from "lodash-es/isEmpty";
import { Project, ProjectCreate } from "../types";
import { projectSlug, randomString } from "../utils";
import { useI18n } from "vue-i18n";

interface LocalState {
  project: ProjectCreate;
}

export default {
  name: "ProjectCreate",
  props: {},
  emits: ["dismiss"],
  setup(props, { emit }) {
    const store = useStore();
    const router = useRouter();
    const { t } = useI18n();

    const state = reactive<LocalState>({
      project: {
        name: "New Project",
        key: randomString(3).toUpperCase(),
      },
    });

    const keyboardHandler = (e: KeyboardEvent) => {
      if (e.code == "Escape") {
        emit("dismiss");
      }
    };

    onMounted(() => {
      document.addEventListener("keydown", keyboardHandler);
    });

    onUnmounted(() => {
      document.removeEventListener("keydown", keyboardHandler);
    });

    const allowCreate = computed(() => {
      return !isEmpty(state.project?.name);
    });

    const create = () => {
      store
        .dispatch("project/createProject", state.project)
        .then((createdProject: Project) => {
          store.dispatch("uistate/saveIntroStateByKey", {
            key: "project.visit",
            newState: true,
          });

          store.dispatch("notification/pushNotification", {
            module: "bytebase",
            style: "SUCCESS",
            title: t("project.create-modal.success-prompt", {
              name: createdProject.name,
            }),
          });

          router.push(`/project/${projectSlug(createdProject)}`);
          emit("dismiss");
        });
    };

    const cancel = () => {
      emit("dismiss");
    };

    return {
      state,
      allowCreate,
      cancel,
      create,
    };
  },
};
</script>
