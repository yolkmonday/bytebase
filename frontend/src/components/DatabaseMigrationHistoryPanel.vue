<template>
  <div class="flex flex-col space-y-4">
    <div
      class="
        flex flex-row
        items-center
        text-lg
        leading-6
        font-medium
        text-main
        space-x-2
      "
    >
      {{ $t('migration-history.self') }}
      <button
        v-if="allowEdit"
        type="button"
        class="ml-4 btn-primary"
        :disabled="state.migrationSetupStatus != 'OK'"
        @click.prevent="state.showBaselineModal = true"
      >
        {{ $t('migration-history.establish-baseline') }}
      </button>
      <div>
        <BBSpin v-if="state.loading" :title="$t('migration-history.refreshing-history')" />
      </div>
    </div>
    <MigrationHistoryTable
      v-if="state.migrationSetupStatus == 'OK'"
      :database-section-list="[database]"
      :history-section-list="migrationHistorySectionList"
    />
    <BBAttention
      v-else
      :style="`WARN`"
      :title="attentionTitle"
      :action-text="allowConfigInstance ? $t('migration-history.config-instance') : ''"
      @click-action="configInstance"
    />
  </div>

  <BBAlert
    v-if="state.showBaselineModal"
    :style="'INFO'"
    :ok-text="$t('migration-history.establish-baseline')"
    :cancel-text="$t('common.cancel')"
    :title="$t('migration-history.establish-database-baseline', { name: database.name })"
    :description="$t('migration-history.establish-baseline-description')"
    @ok="
      () => {
        doCreateBaseline();
      }
    "
    @cancel="state.showBaselineModal = false"
  >
  </BBAlert>
</template>

<script lang="ts">
import { computed, PropType, reactive, watchEffect } from "vue";
import { useStore } from "vuex";
import MigrationHistoryTable from "../components/MigrationHistoryTable.vue";
import {
  Database,
  InstanceMigration,
  MigrationHistory,
  MigrationSchemaStatus,
} from "../types";
import { useRouter } from "vue-router";
import { BBTableSectionDataSource } from "../bbkit/types";
import { instanceSlug, isDBAOrOwner } from "../utils";
import { useI18n } from "vue-i18n";

interface LocalState {
  migrationSetupStatus: MigrationSchemaStatus;
  showBaselineModal: boolean;
  loading: boolean;
}

export default {
  name: "DatabaseMigrationHistoryPanel",
  components: { MigrationHistoryTable },
  props: {
    database: {
      required: true,
      type: Object as PropType<Database>,
    },
    allowEdit: {
      required: true,
      type: Boolean,
    },
  },
  setup(props) {
    const { t } = useI18n();

    const store = useStore();
    const router = useRouter();

    const state = reactive<LocalState>({
      migrationSetupStatus: "OK",
      showBaselineModal: false,
      loading: false,
    });

    const currentUser = computed(() => store.getters["auth/currentUser"]());

    const prepareMigrationHistoryList = () => {
      state.loading = true;
      store
        .dispatch("instance/checkMigrationSetup", props.database.instance.id)
        .then((migration: InstanceMigration) => {
          state.migrationSetupStatus = migration.status;
          if (state.migrationSetupStatus == "OK") {
            store
              .dispatch("instance/fetchMigrationHistory", {
                instanceId: props.database.instance.id,
                databaseName: props.database.name,
              })
              .then(() => {
                state.loading = false;
              })
              .catch(() => {
                state.loading = false;
              });
          }
        })
        .catch(() => {
          state.loading = false;
        });
    };

    watchEffect(prepareMigrationHistoryList);

    const isCurrentUserDBAOrOwner = computed((): boolean => {
      return isDBAOrOwner(currentUser.value.role);
    });

    const allowConfigInstance = computed(() => {
      return isCurrentUserDBAOrOwner.value;
    });

    const attentionTitle = computed((): string => {
      if (state.migrationSetupStatus == "NOT_EXIST") {
        return (
          t('migration-history.instance-missing-migration-schema', { name: props.database.instance.name }) +
          (isDBAOrOwner(currentUser.value.role)
            ? ""
            : " " + t('migration-history.contact-dba'))
        );
      } else if (state.migrationSetupStatus == "UNKNOWN") {
        return (
          t('migration-history.instance-bad-connection', { name: props.database.instance.name })  +
          (isDBAOrOwner(currentUser.value.role)
            ? ""
            : " " + t('migration-history.contact-dba'))
        );
      }
      return "";
    });

    const migrationHistorySectionList = computed(
      (): BBTableSectionDataSource<MigrationHistory>[] => {
        return [
          {
            title: "",
            list: store.getters[
              "instance/migrationHistoryListByInstanceIdAndDatabaseName"
            ](props.database.instance.id, props.database.name),
          },
        ];
      }
    );

    const configInstance = () => {
      router.push(`/instance/${instanceSlug(props.database.instance)}`);
    };

    const doCreateBaseline = () => {
      state.showBaselineModal = false;
      router.push({
        name: "workspace.issue.detail",
        params: {
          issueSlug: "new",
        },
        query: {
          template: "bb.issue.database.schema.baseline",
          name: t('migration-history.establish-database-baseline', { name: props.database.name }),
          project: props.database.project.id,
          databaseList: `${props.database.id}`,
        },
      });
    };

    return {
      state,
      allowConfigInstance,
      attentionTitle,
      migrationHistorySectionList,
      configInstance,
      doCreateBaseline,
    };
  },
};
</script>
