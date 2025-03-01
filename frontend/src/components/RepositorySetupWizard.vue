<template>
  <div>
    <div class="textinfolabel">
      <i18n-t keypath="repository.setup-wizard-guide">
        <template #guide>
          <a
            href="https://docs.bytebase.com/use-bytebase/vcs-integration/link-repository?ref=console"
            target="_blank"
            class="normal-link"
          >
            {{ $t("common.detailed-guide") }}</a
          >
        </template>
      </i18n-t>
    </div>
    <BBStepTab
      class="pt-4"
      :step-item-list="stepList"
      :allow-next="allowNext"
      @try-change-step="tryChangeStep"
      @try-finish="tryFinishSetup"
      @cancel="cancel"
    >
      <template #0="{ next }">
        <RepositoryVCSProviderPanel :config="state.config" @next="next()" />
      </template>
      <template #1="{ next }">
        <RepositorySelectionPanel :config="state.config" @next="next()" />
      </template>
      <template #2>
        <RepositoryConfigPanel :config="state.config" />
      </template>
    </BBStepTab>
  </div>
</template>

<script lang="ts">
import { reactive, computed, PropType } from "vue";
import { useStore } from "vuex";
import { useRouter } from "vue-router";
import isEmpty from "lodash-es/isEmpty";
import { BBStepTabItem } from "../bbkit/types";
import RepositoryVCSProviderPanel from "./RepositoryVCSProviderPanel.vue";
import RepositorySelectionPanel from "./RepositorySelectionPanel.vue";
import RepositoryConfigPanel from "./RepositoryConfigPanel.vue";
import {
  Project,
  ProjectRepositoryConfig,
  RepositoryCreate,
  unknown,
  VCS,
} from "../types";
import { projectSlug } from "../utils";
import { useI18n } from "vue-i18n";

// Default file path template is to organize migration files from different environments under separate directories.
const DEFAULT_FILE_PATH_TEMPLATE =
  "{{ENV_NAME}}/{{DB_NAME}}__{{VERSION}}__{{TYPE}}__{{DESCRIPTION}}.sql";
// Default schema path template is co-locate with the corresponding db's migration files and use .(dot) to appear the first.
const DEFAULT_SCHEMA_PATH_TEMPLATE = "{{ENV_NAME}}/.{{DB_NAME}}__LATEST.sql";

const CHOOSE_PROVIDER_STEP = 0;
// const CHOOSE_REPOSITORY_STEP = 1;
const CONFIGURE_DEPLOY_STEP = 2;

interface LocalState {
  config: ProjectRepositoryConfig;
  currentStep: number;
}

export default {
  name: "RepositorySetupWizard",
  components: {
    RepositoryVCSProviderPanel,
    RepositorySelectionPanel,
    RepositoryConfigPanel,
  },
  props: {
    // If false, then we intend to change the existing linked repository intead of just linking a new repository.
    create: {
      type: Boolean,
      default: false,
    },
    project: {
      required: true,
      type: Object as PropType<Project>,
    },
  },
  emits: ["cancel", "finish"],
  setup(props, { emit }) {
    const { t } = useI18n();

    const router = useRouter();
    const store = useStore();

    const stepList: BBStepTabItem[] = [
      { title: t("repository.choose-git-provider"), hideNext: true },
      { title: t("repository.select-repository"), hideNext: true },
      { title: t("repository.configure-deploy") },
    ];

    const state = reactive<LocalState>({
      config: {
        vcs: unknown("VCS") as VCS,
        code: "",
        token: {
          accessToken: "",
          expiresTs: 0,
          refreshToken: "",
        },
        repositoryInfo: {
          externalId: "",
          name: "",
          fullPath: "",
          webUrl: "",
        },
        repositoryConfig: {
          baseDirectory: "",
          branchFilter: "",
          filePathTemplate: DEFAULT_FILE_PATH_TEMPLATE,
          schemaPathTemplate: DEFAULT_SCHEMA_PATH_TEMPLATE,
        },
      },
      currentStep: CHOOSE_PROVIDER_STEP,
    });

    const allowNext = computed((): boolean => {
      if (state.currentStep == CONFIGURE_DEPLOY_STEP) {
        return (
          !isEmpty(state.config.repositoryConfig.branchFilter.trim()) &&
          !isEmpty(state.config.repositoryConfig.filePathTemplate.trim())
        );
      }
      return true;
    });

    const tryChangeStep = (
      oldStep: number,
      newStep: number,
      allowChangeCallback: () => void
    ) => {
      state.currentStep = newStep;
      allowChangeCallback();
    };

    const tryFinishSetup = (allowFinishCallback: () => void) => {
      const createFunc = () => {
        const repositoryCreate: RepositoryCreate = {
          vcsId: state.config.vcs.id,
          projectId: props.project.id,
          name: state.config.repositoryInfo.name,
          fullPath: state.config.repositoryInfo.fullPath,
          webUrl: state.config.repositoryInfo.webUrl,
          branchFilter: state.config.repositoryConfig.branchFilter,
          baseDirectory: state.config.repositoryConfig.baseDirectory,
          filePathTemplate: state.config.repositoryConfig.filePathTemplate,
          schemaPathTemplate: state.config.repositoryConfig.schemaPathTemplate,
          externalId: state.config.repositoryInfo.externalId,
          accessToken: state.config.token.accessToken,
          expiresTs: state.config.token.expiresTs,
          refreshToken: state.config.token.refreshToken,
        };
        store
          .dispatch("repository/createRepository", repositoryCreate)
          .then(() => {
            allowFinishCallback();
            emit("finish");
          });
      };
      if (props.create) {
        createFunc();
      } else {
        // It's simple to implement change behavior as delete followed by create.
        // Though the delete can succeed while the create fails, this is rare, and
        // even it happens, user can still configure it again.
        store
          .dispatch("repository/deleteRepositoryByProjectId", props.project.id)
          .then(() => {
            createFunc();
          });
      }
    };

    const cancel = () => {
      emit("cancel");
      router.push({
        name: "workspace.project.detail",
        params: {
          projectSlug: projectSlug(props.project),
        },
        hash: "#version-control",
      });
    };

    return {
      state,
      stepList,
      allowNext,
      tryChangeStep,
      tryFinishSetup,
      cancel,
    };
  },
};
</script>
