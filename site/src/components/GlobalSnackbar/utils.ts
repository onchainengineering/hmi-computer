import { dispatchCustomEvent } from "../../util/events"

///////////////////////////////////////////////////////////////////////////////
// Notification Types
///////////////////////////////////////////////////////////////////////////////

export enum MsgType {
  Info,
  Success,
  Error,
}

/**
 * Display a prefixed paragraph inside a notification.
 */
export type NotificationTextPrefixed = {
  prefix: string
  text: string
}

export type AdditionalMessage = NotificationTextPrefixed | string[] | string

export const isNotificationText = (msg: AdditionalMessage): msg is string => {
  return !Array.isArray(msg) && typeof msg === "string"
}

export const isNotificationTextPrefixed = (msg: AdditionalMessage | null): msg is NotificationTextPrefixed => {
  // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
  return typeof (msg as NotificationTextPrefixed)?.prefix !== "undefined"
}

export const isNotificationList = (msg: AdditionalMessage): msg is string[] => {
  return Array.isArray(msg)
}

export interface NotificationMsg {
  msgType: MsgType
  msg: string
  additionalMsgs?: AdditionalMessage[]
}

export const SnackbarEventType = "coder:notification"

///////////////////////////////////////////////////////////////////////////////
// Notification Functions
///////////////////////////////////////////////////////////////////////////////

function dispatchNotificationEvent(msgType: MsgType, msg: string, additionalMsgs?: AdditionalMessage[]) {
  dispatchCustomEvent<NotificationMsg>(SnackbarEventType, {
    msgType,
    msg,
    additionalMsgs,
  })
}

export const displayMsg = (msg: string, additionalMsg?: string): void => {
  dispatchNotificationEvent(MsgType.Info, msg, additionalMsg ? [additionalMsg] : undefined)
}

export const displaySuccess = (msg: string, additionalMsg?: string): void => {
  dispatchNotificationEvent(MsgType.Success, msg, additionalMsg ? [additionalMsg] : undefined)
}
