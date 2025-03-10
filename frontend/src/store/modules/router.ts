import { RouteLocationNormalized } from "vue-router";
import { RouterSlug } from "../../types";

// eslint-disable-next-line @typescript-eslint/no-empty-interface
export interface RouterState {}

const getters = {
  backPath: (state: RouterState) => (): string => {
    return localStorage.getItem("ui.backPath") || "/";
  },

  routeSlug:
    (state: RouterState) =>
    (currentRoute: RouteLocationNormalized): RouterSlug => {
      {
        // /u/:principalId
        // Total 2 elements, 2nd element is the principal id
        const profileComponents = currentRoute.path.match(
          "/u/([0-9a-zA-Z_-]+)"
        ) || ["/", undefined];
        if (profileComponents[1]) {
          return {
            principalId: parseInt(profileComponents[1]),
          };
        }
      }

      {
        // /environment/:environmentSlug
        // Total 2 elements, 2nd element is the issue slug
        const environmentComponents = currentRoute.path.match(
          "/environment/([0-9a-zA-Z_-]+)"
        ) || ["/", undefined];
        if (environmentComponents[1]) {
          return {
            environmentSlug: environmentComponents[1],
          };
        }
      }

      {
        // /project/:projectSlug/webhook/:hookSlug
        // Total 3 elements, 2nd element is the project slug, 3rd element is the project webhook slug
        const projectComponents = currentRoute.path.match(
          "/project/([0-9a-zA-Z_-]+)/webhook/([0-9a-zA-Z_-]+)"
        ) || ["/", undefined, undefined];
        if (projectComponents[1] && projectComponents[2]) {
          return {
            projectSlug: projectComponents[1],
            projectWebhookSlug:
              projectComponents[2].toLowerCase() == "new"
                ? undefined
                : projectComponents[2],
          };
        }
      }

      {
        // /project/:projectSlug
        // Total 2 elements, 2nd element is the project slug
        const projectComponents = currentRoute.path.match(
          "/project/([0-9a-zA-Z_-]+)"
        ) || ["/", undefined];
        if (projectComponents[1]) {
          return {
            projectSlug: projectComponents[1],
          };
        }
      }

      {
        // /issue/:issueSlug
        // Total 2 elements, 2nd element is the issue slug
        const issueComponents = currentRoute.path.match(
          "/issue/([0-9a-zA-Z_-]+)"
        ) || ["/", undefined];
        if (issueComponents[1]) {
          return {
            issueSlug: issueComponents[1],
          };
        }
      }

      {
        // /db/:databaseSlug/table/:tableName
        // Total 3 elements, 2nd element is the database slug, 3rd element is the table name
        const databaseComponents = currentRoute.path.match(
          "/db/([0-9a-zA-Z_-]+)/table/(.+)"
        ) || ["/", undefined, undefined];
        if (databaseComponents[1] && databaseComponents[2]) {
          return {
            databaseSlug: databaseComponents[1],
            tableName: databaseComponents[2],
          };
        }
      }

      {
        // /db/:databaseSlug/datasource/:dataSourceSlug
        // Total 3 elements, 2nd element is the database slug, 3rd element is the data source slug
        const dataSourceComponents = currentRoute.path.match(
          "/db/([0-9a-zA-Z_-]+)/datasource/([0-9a-zA-Z_-]+)"
        ) || ["/", undefined, undefined];
        if (dataSourceComponents[1] && dataSourceComponents[2]) {
          return {
            databaseSlug: dataSourceComponents[1],
            dataSourceSlug: dataSourceComponents[2],
          };
        }
      }

      {
        // /db/:databaseSlug/history/:migrationHistorySlug
        // Total 3 elements, 2nd element is the database slug, 3rd element is the migration history slug
        const migrationHistoryComponents = currentRoute.path.match(
          "/db/([0-9a-zA-Z_-]+)/history/([0-9a-zA-Z_-]+)"
        ) || ["/", undefined, undefined];
        if (migrationHistoryComponents[1] && migrationHistoryComponents[2]) {
          return {
            databaseSlug: migrationHistoryComponents[1],
            migrationHistorySlug: migrationHistoryComponents[2],
          };
        }
      }

      {
        // /db/:databaseSlug
        // Total 2 elements, 2nd element is the database slug
        const databaseComponents = currentRoute.path.match(
          "/db/([0-9a-zA-Z_-]+)"
        ) || ["/", undefined];
        if (databaseComponents[1]) {
          return {
            databaseSlug: databaseComponents[1],
          };
        }
      }

      {
        // /instance/:instanceSlug
        // Total 2 elements, 2nd element is the instance slug
        const instanceComponents = currentRoute.path.match(
          "/instance/([0-9a-zA-Z_-]+)"
        ) || ["/", undefined];
        if (instanceComponents[1]) {
          return {
            instanceSlug: instanceComponents[1],
          };
        }
      }

      {
        // /setting/version-control/:vcsId
        // Total 2 elements, 2nd element is the version control system id
        const vcsComponents = currentRoute.path.match(
          "/setting/version-control/([0-9a-zA-Z_-]+)"
        ) || ["/", undefined];
        if (vcsComponents[1]) {
          return {
            vcsSlug: vcsComponents[1],
          };
        }
      }

      return {};
    },
};

const actions = {
  setBackPath({ commit }: any, backPath: string) {
    localStorage.setItem("ui.backPath", backPath);
    return backPath;
  },
};

export default {
  namespaced: true,
  getters,
  actions,
};
