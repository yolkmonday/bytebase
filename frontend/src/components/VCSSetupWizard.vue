<template>
  <BBAttention
    v-if="showAttention"
    :style="'WARN'"
    :description="attentionText"
  />
  <BBStepTab
    class="mt-4"
    :step-item-list="stepList"
    :allow-next="allowNext"
    :finish-title="'Confirm and add'"
    @try-change-step="tryChangeStep"
    @try-finish="tryFinishSetup"
    @cancel="cancelSetup"
  >
    <template #0>
      <VCSProviderBasicInfoPanel :config="state.config" />
    </template>
    <template #1>
      <VCSProviderOAuthPanel :config="state.config" />
    </template>
    <template #2>
      <VCSProviderConfirmPanel :config="state.config" />
    </template>
  </BBStepTab>
</template>

<script lang="ts">
import { reactive, computed } from "vue";
import { useRouter } from "vue-router";
import { useStore } from "vuex";
import isEmpty from "lodash-es/isEmpty";
import { BBStepTabItem } from "../bbkit/types";
import VCSProviderBasicInfoPanel from "./VCSProviderBasicInfoPanel.vue";
import VCSProviderOAuthPanel from "./VCSProviderOAuthPanel.vue";
import VCSProviderConfirmPanel from "./VCSProviderConfirmPanel.vue";
import {
  isValidVCSApplicationIdOrSecret,
  VCSConfig,
  VCSCreate,
  VCS,
  openWindowForOAuth,
  OAuthWindowEventPayload,
  OAuthWindowEvent,
  OAuthConfig,
  redirectUrl,
  OAuthToken,
} from "../types";
import { isUrl } from "../utils";
import { useI18n } from "vue-i18n";

const BASIC_INFO_STEP = 0;
const OAUTH_INFO_STEP = 1;
const CONFIRM_STEP = 2;

interface LocalState {
  config: VCSConfig;
  currentStep: number;
  oAuthResultCallback?: (token: OAuthToken | undefined) => void;
}

export default {
  name: "VCSSetupWizard",
  components: {
    VCSProviderBasicInfoPanel,
    VCSProviderOAuthPanel,
    VCSProviderConfirmPanel,
  },
  setup() {
    const { t } = useI18n();
    const store = useStore();
    const router = useRouter();

    const stepList: BBStepTabItem[] = [
      { title: t("version-control.setting.add-git-provider.basic-info.self") },
      { title: t("version-control.setting.add-git-provider.oauth-info.self") },
      { title: t("common.confirm") },
    ];

    const state = reactive<LocalState>({
      config: {
        type: "GITLAB_SELF_HOST",
        name: t("version-control.setting.add-git-provider.gitlab-self-host"),
        instanceUrl: "",
        applicationId: "",
        secret: "",
      },
      currentStep: 0,
    });

    const eventListener = (event: Event) => {
      const payload = (event as CustomEvent).detail as OAuthWindowEventPayload;
      if (isEmpty(payload.error)) {
        if (state.config.type == "GITLAB_SELF_HOST") {
          const oAuthConfig: OAuthConfig = {
            endpoint: `${state.config.instanceUrl}/oauth/token`,
            applicationId: state.config.applicationId,
            secret: state.config.secret,
            redirectUrl: redirectUrl(),
          };
          store
            .dispatch("gitlab/exchangeToken", {
              oAuthConfig,
              code: payload.code,
            })
            .then((token: OAuthToken) => {
              state.oAuthResultCallback!(token);
            })
            .catch(() => {
              state.oAuthResultCallback!(undefined);
            });
        }
      } else {
        state.oAuthResultCallback!(undefined);
      }

      window.removeEventListener(OAuthWindowEvent, eventListener);
    };

    const allowNext = computed((): boolean => {
      if (state.currentStep == BASIC_INFO_STEP) {
        return isUrl(state.config.instanceUrl);
      } else if (state.currentStep == OAUTH_INFO_STEP) {
        return (
          isValidVCSApplicationIdOrSecret(state.config.applicationId) &&
          isValidVCSApplicationIdOrSecret(state.config.secret)
        );
      }
      return true;
    });

    const attentionText = computed((): string => {
      if (state.config.type == "GITLAB_SELF_HOST") {
        return t(
          "version-control.setting.add-git-provider.gitlab-self-host-admin-requirement"
        );
      }
      return "";
    });

    const showAttention = computed((): boolean => {
      return state.currentStep != CONFIRM_STEP;
    });

    const tryChangeStep = (
      oldStep: number,
      newStep: number,
      allowChangeCallback: () => void
    ) => {
      // If we are trying to move from OAuth step to Confirm step, we first verify
      // the OAuth info is correct. We achieve this by:
      // 1. Kicking of the OAuth workflow to verify the current user can login to the GitLab instance and the application id is correct.
      // 2. If step 1 succeeds, we will get a code, we use this code together with the secret to exchange for the access token. (see eventListener)
      if (state.currentStep == OAUTH_INFO_STEP && newStep > oldStep) {
        const newWindow = openWindowForOAuth(
          `${state.config.instanceUrl}/oauth/authorize`,
          state.config.applicationId
        );
        if (newWindow) {
          state.oAuthResultCallback = (token: OAuthToken | undefined) => {
            if (token) {
              state.currentStep = newStep;
              allowChangeCallback();
              store.dispatch("notification/pushNotification", {
                module: "bytebase",
                style: "SUCCESS",
                title: t(
                  "version-control.setting.add-git-provider.ouath-info-correct"
                ),
              });
            } else {
              var description = "";
              if (state.config.type == "GITLAB_SELF_HOST") {
                // If application id mismatches, the OAuth workflow will stop early.
                // So the only possibility to reach here is we have a matching application id, while
                // we failed to exchange a token, and it's likely we are requesting with a wrong secret.
                description = t(
                  "version-control.setting.add-git-provider.check-oauth-info-match"
                );
              }
              store.dispatch("notification/pushNotification", {
                module: "bytebase",
                style: "CRITICAL",
                title: "Failed to setup OAuth",
                description: description,
              });
            }
          };
          window.addEventListener(OAuthWindowEvent, eventListener, false);
        }
      } else {
        state.currentStep = newStep;
        allowChangeCallback();
      }
    };

    const tryFinishSetup = (allowChangeCallback: () => void) => {
      const vcsCreate: VCSCreate = {
        ...state.config,
      };
      store.dispatch("vcs/createVCS", vcsCreate).then((vcs: VCS) => {
        allowChangeCallback();
        router.push({
          name: "setting.workspace.version-control",
        });
        store.dispatch("notification/pushNotification", {
          module: "bytebase",
          style: "SUCCESS",
          title: t("version-control.setting.add-git-provider.add-success", [
            vcs.name,
          ]),
        });
      });
    };

    const cancelSetup = () => {
      router.push({
        name: "setting.workspace.version-control",
      });
    };

    return {
      stepList,
      state,
      allowNext,
      attentionText,
      showAttention,
      tryChangeStep,
      tryFinishSetup,
      cancelSetup,
    };
  },
};
</script>
