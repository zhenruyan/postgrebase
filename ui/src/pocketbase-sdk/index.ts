import Client from '@sdk/Client';
import type {
    SendOptions,
    BeforeSendResult,
} from '@sdk/Client';
import ClientResponseError from '@sdk/ClientResponseError';
import ExternalAuth        from '@sdk/models/ExternalAuth';
import Admin               from '@sdk/models/Admin';
import Collection          from '@sdk/models/Collection';
import Record              from '@sdk/models/Record';
import LogRequest          from '@sdk/models/LogRequest';
import BaseModel           from '@sdk/models/utils/BaseModel';
import ListResult          from '@sdk/models/utils/ListResult';
import SchemaField         from '@sdk/models/utils/SchemaField';
import CrudService         from '@sdk/services/utils/CrudService';
import AdminService        from '@sdk/services/AdminService';
import CollectionService   from '@sdk/services/CollectionService';
import LogService          from '@sdk/services/LogService';
import RealtimeService     from '@sdk/services/RealtimeService';
import RecordService       from '@sdk/services/RecordService';
import SettingsService     from '@sdk/services/SettingsService';
import LocalAuthStore      from '@sdk/stores/LocalAuthStore';
import {
    getTokenPayload,
    isTokenExpired,
} from '@sdk/stores/utils/jwt';
import BaseAuthStore from '@sdk/stores/BaseAuthStore';
import type {
    OnStoreChangeFunc,
} from '@sdk/stores/BaseAuthStore';
import type {
    RecordAuthResponse,
    AuthProviderInfo,
    AuthMethodsList,
    RecordSubscription,
    OAuth2UrlCallback,
    OAuth2AuthConfig,
} from '@sdk/services/RecordService';
import type { UnsubscribeFunc } from '@sdk/services/RealtimeService';
import type { BackupFileInfo } from '@sdk/services/BackupService';
import type { HealthCheckResponse } from '@sdk/services/HealthService';
import type {
    BaseQueryParams,
    ListQueryParams,
    RecordQueryParams,
    RecordListQueryParams,
    LogStatsQueryParams,
    FileQueryParams,
    FullListQueryParams,
    RecordFullListQueryParams,
} from '@sdk/services/utils/QueryParams';

export {
    ClientResponseError,
    BaseAuthStore,
    LocalAuthStore,
    getTokenPayload,
    isTokenExpired,
    ExternalAuth,
    Admin,
    Collection,
    Record,
    LogRequest,
    BaseModel,
    ListResult,
    SchemaField,

    // services
    CrudService,
    AdminService,
    CollectionService,
    LogService,
    RealtimeService,
    RecordService,
    SettingsService,
};

export type {
    HealthCheckResponse,
    BackupFileInfo,
    SendOptions,
    BeforeSendResult,
    RecordAuthResponse,
    AuthProviderInfo,
    AuthMethodsList,
    RecordSubscription,
    OAuth2UrlCallback,
    OAuth2AuthConfig,
    OnStoreChangeFunc,
    UnsubscribeFunc,
    BaseQueryParams,
    ListQueryParams,
    RecordQueryParams,
    RecordListQueryParams,
    LogStatsQueryParams,
    FileQueryParams,
    FullListQueryParams,
    RecordFullListQueryParams,
};

export default Client;
