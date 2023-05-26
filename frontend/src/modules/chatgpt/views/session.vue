<template>
	<cl-crud ref="Crud">
		<cl-row>
			<!-- 刷新按钮 -->
			<cl-refresh-btn />
			<!-- 新增按钮 -->
			<cl-add-btn />
			<!-- 删除按钮 -->
			<cl-multi-delete-btn />
			<cl-flex1 />
			<!-- 关键字搜索 -->
			<cl-search-key />
		</cl-row>

		<cl-row>
			<!-- 数据表格 -->
			<cl-table ref="Table" />
		</cl-row>

		<cl-row>
			<cl-flex1 />
			<!-- 分页控件 -->
			<cl-pagination />
		</cl-row>

		<!-- 新增、编辑 -->
		<cl-upsert ref="Upsert" />
	</cl-crud>
</template>

<script lang="ts" name="chatgpt-session" setup>
import { useCrud, useTable, useUpsert } from "@cool-vue/crud";
import { useCool } from "/@/cool";

const { service } = useCool();

// cl-upsert 配置
const Upsert = useUpsert({
	items: [
		{ label: "邮箱", prop: "email", required: true, component: { name: "el-input" } },
		{ label: "密码", prop: "password", required: true, component: { name: "el-input" } },
		{
			label: "状态",
			prop: "status",
			component: {
				name: "el-switch",
				props: {
					activeValue: 1,
					inactiveValue: 0
				}
			}
		},
		{
			label: "PLUS",
			prop: "isPlus",
			component: {
				name: "el-switch",
				props: {
					activeValue: 1,
					inactiveValue: 0
				}
			}
		},
		{
			label: "官方session",
			prop: "officialSession",
			component: { name: "el-input", props: { type: "textarea", rows: 4 } }
		},
		{
			label: "备注",
			prop: "remark",
			component: { name: "el-input", props: { type: "textarea", rows: 4 } }
		}
	]
});

// cl-table 配置
const Table = useTable({
	columns: [
		{ type: "selection" },
		{ label: "id", prop: "id" },
		{ label: "创建时间", prop: "createTime" },
		{ label: "更新时间", prop: "updateTime" },
		{ label: "邮箱", prop: "email" },
		{ label: "密码", prop: "password" },
		{ label: "状态", prop: "status", component: { name: "cl-switch" } },
		{ label: "PLUS", prop: "isPlus", component: { name: "cl-switch" } },
		{ label: "官方session", prop: "officialSession", showOverflowTooltip: true },
		{ label: "备注", prop: "remark", showOverflowTooltip: true },
		{ type: "op", buttons: ["edit", "delete"] }
	]
});

// cl-crud 配置
const Crud = useCrud(
	{
		service: service.chatgpt.session
	},
	(app) => {
		app.refresh();
	}
);
</script>
