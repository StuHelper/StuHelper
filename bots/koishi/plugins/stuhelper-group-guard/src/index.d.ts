import { Context, Schema } from 'koishi';
import { type StuhelperGroupGuardPluginConfig } from '@stuhelper/koishi-shared';
export declare const name = "stuhelper-group-guard";
export type Config = StuhelperGroupGuardPluginConfig;
export declare const Config: Schema<Config>;
export declare function apply(ctx: Context, config: Config): void;
