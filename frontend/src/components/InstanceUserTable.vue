<template>
  <BBTable
    :column-list="COLUMN_LIST"
    :data-source="instanceUserList"
    :show-header="true"
    :row-clickable="false"
    :left-bordered="true"
    :right-bordered="true"
  >
    <template #body="{ rowData: instanceUser }">
      <BBTableCell :left-padding="4" class="w-4">
        {{ instanceUser.name }}
      </BBTableCell>
      <BBTableCell class="whitespace-pre-wrap">
        {{ instanceUser.grant.replaceAll("\n", "\n\n") }}
      </BBTableCell>
    </template>
  </BBTable>
</template>

<script lang="ts">
import { PropType } from "vue";
import { BBTableColumn } from "../bbkit/types";
import { InstanceUser } from "../types/InstanceUser";
import { useI18n } from "vue-i18n";

export default {
  name: "InstanceUserTable",
  components: {},
  props: {
    instanceUserList: {
      required: true,
      type: Object as PropType<InstanceUser[]>,
    },
  },
  setup() {
    const { t } = useI18n();
    const COLUMN_LIST: BBTableColumn[] = [
      {
        title: t("common.User"),
      },
      {
        title: t("instance.grants"),
      },
    ];
    return {
      COLUMN_LIST,
    };
  },
};
</script>
