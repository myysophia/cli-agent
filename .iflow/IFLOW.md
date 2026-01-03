# Project Context for iFlow CLI

## Project Type
This directory appears to be a minimal setup for an iFlow CLI context. It contains only the `IFLOW.md` file.

## Directory Overview
The directory contains a single file:
- `IFLOW.md`: This file, which serves as the context for iFlow CLI interactions.

## Key Files
- `IFLOW.md`: The primary file that defines the project context and instructions for iFlow CLI.

## Usage
This directory is intended to be used as a context for iFlow CLI interactions. The `IFLOW.md` file provides the necessary information for the CLI to understand the project environment.

## RULE
当用户的需求是生成一个调研或分析报告时，请遵循以下规范

（1）任何报告都需要有框架描述文件：
- 描述文件应该包括：报告标题和分析目标、整个报告包括哪些section、每个section包括的大致内容以及subsection细分、每个section中需要进行可视化分析的数据图表。
- 框架描述文件的具体设计需要严格遵循用户输入或已有的文件中提供的框架设计要求。如果用户没有提供，且文件中没有相关设计信息，请你根据已有的文件自行设计，并存储在文件夹中。
- 框架描述文件应以markdown格式存储，按section顺序组织内容

（2）现在我有一些subagent，下面是每个agent对应的能力scope：
-   1.format_md_agent：markdown报告编写专家，可以初始化markdown报告的框架，也可以向markdown文件中固定模块添加内容、维护项目。任务描述中，需要详细描述任务需求。如果需要初始化框架，请告诉我整个报告需要使用的文件路径和框架描述文件；如果需要填充已有的报告，请告诉我需要填充的章节，以及本次填充需要使用的部分文件路径。本专家没有数据分析的能力，如果需要在报告中表达专业的数据分析内容、图表等，请提前生成好描述在任务文件中。本专家也没有数据收集能力，请告诉我尽可能多的数据收集结果文件。最终项目会产出在工作空间下。
-   2.format_html_agent：html报告编写专家，可以初始化html报告的框架，也可以向html文件中固定模块添加内容、维护项目。任务描述中，需要详细描述任务需求。如果需要初始化框架，请告诉我整个报告需要使用的文件路径和框架描述文件；如果需要填充已有的报告，请告诉我需要填充的section，以及本次填充需要使用的部分文件路径。本专家没有数据分析的能力，如果需要在报告中表达专业的数据分析内容、图表等，请提前生成好描述在任务文件中。本专家也没有数据收集能力，请告诉我尽可能多的数据收集结果文件。最终项目会产出在工作空间下。
-   3.perception_agent：感知到已经收集到的商品信息，对已有的文件结构做初步的分析
-   4.data_analysis_agent：专业数据分析师，善于数据标注与数据可视化，可以使用pyecharts绘制可视化图表。
-   5.data_collection_agent：专业的“数据信息收集者”，专注于从全网搜集、甄别与整合信息的采集代理，支持多源检索、去重汇总与来源标注，输出信息搜索结果并保存到工作目录。该agent支持并行调用，可将拆分后的搜索任务下发并行执行，提高效率。建议并行执行的搜索子任务不能超过3个。

（3）当需要完成调研分析报告的时候，整理流程请参考以下步骤，使用TODO list来维护整个流程：
-    0.根据用户需求，确定最终生成的报告是html还是md格式；如果用户没有指出，默认使用md格式。报告格式决定了后续的Format Agent是format_md_agent还是format_html_agent，一个任务只能使用一种Format Agent
-    1.调用perception_agent，感知目前已有的文件信息
-    2.根据用户需求和已有的文件信息，生成一份报告框架描述文件。注意，生成这个文件不等价于Format Agent初始化报告，这个文件需要由你生成。
-    3.调用Format Agent初始化报告，需要给出报告框架的描述文件位置，不能指定最终报告存储位置和名字
-    4.调用data_collection_agent，补充搜索生成报告缺失的信息。
-    5.逐个section的填充报告，每个section的填充顺序是（1）如果这个section需要添加图表，先调用data_analysis_agent进行可视化分析，绘制这个section需要的图表（2）调用Format Agent填充section的内容。第5步需要重复执行，直到所有section都被填充结束。
-    #### 第5步必须按section分发任务，不允许将多个section合并填充，因为其他agent能力不足以执行这么多任务。
-    #### 注意：上述流程每一步必须分成多次agent调用，不能在一次agent调用，所以需要分成多个task指定agent任务
-    6.向用户展示报告：
    - 如果你的报告是html格式的，请调用upload_folder_to_oss工具，使用绝对路径上传"reporter_agent"到oss。根据返回的url链接，调用show_report展示html的url链接
    - 如果你的报告是md格式的，请直接调用show_report，展示md的text内容
