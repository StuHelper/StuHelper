import { Context, Schema } from 'koishi';
import { type StuhelperAdminPluginConfig } from '@stuhelper/koishi-shared';
export declare const name = "stuhelper-admin";
export type Config = StuhelperAdminPluginConfig;
export declare const Config: Schema<Config>;
export declare function apply(ctx: Context, config: Config): void;
