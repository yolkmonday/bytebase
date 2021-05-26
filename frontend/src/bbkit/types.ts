export type BBButtonConfirmStyle = "DELETE" | "ARCHIVE" | "RESTORE";

export type BBTableColumn = {
  title: string;
};

export type BBTableSectionDataSource<T> = {
  title: string;
  link?: string;
  list: T[];
};

export type BBTabItem = {
  title: string;
  // Used as the anchor
  id: string;
};

export type BBStepStatus =
  | "PENDING"
  | "PENDING_ACTIVE"
  | "PENDING_APPROVAL"
  | "PENDING_APPROVAL_ACTIVE"
  | "RUNNING"
  | "DONE"
  | "FAILED"
  | "CANCELED"
  | "SKIPPED";

export type BBStep = {
  title: string;
  status: BBStepStatus;
  link(): string;
};

export type BBOutlineItem = {
  id: string;
  name: string;
  link?: string;
  childList?: BBOutlineItem[];
  // Only applicable if childList is specified.
  childCollapse?: boolean;
};

export type BBNotificationStyle = "INFO" | "SUCCESS" | "WARN" | "CRITICAL";
export type BBNotificationPlacement =
  | "TOP_LEFT"
  | "TOP_RIGHT"
  | "BOTTOM_LEFT"
  | "BOTTOM_RIGHT";
export type BBNotificationItem = {
  style: BBNotificationStyle;
  title: string;
  description: string;
  link: string;
  linkTitle: string;
};

export type BBAlertStyle = "INFO" | "SUCCESS" | "WARN" | "CRITICAL";
