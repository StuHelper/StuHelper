import { Context, Schema } from 'koishi';
import { type StuhelperBindingPluginConfig } from '@stuhelper/koishi-shared';
export declare const name = "stuhelper-binding";
export type Config = StuhelperBindingPluginConfig;
export declare const Config: Schema<Config>;
export declare function apply(ctx: Context, config: Config): void;
