import { SettingId } from "./id";
import { Principal } from "./principal";

export type SettingName = "bb.console.url";

export type Setting = {
  id: SettingId;

  // Standard fields
  creator: Principal;
  createdTs: number;
  updater: Principal;
  updatedTs: number;

  // Domain specific fields
  name: SettingName;
  value: string;
  description: string;
};
