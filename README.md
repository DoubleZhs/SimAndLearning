# SimAndLearning 交通模拟与学习项目

## 项目介绍

SimAndLearning是一个基于元胞自动机的交通模拟系统，能够模拟多种交通场景，包括道路网络、车辆行为和交通灯控制等。该项目实现了纳格尔模型（Nagel-Schreckenberg model）来模拟车辆行为，并提供了丰富的接口以支持扩展和研究。

## 项目结构

```
SimAndLearning/
├── element/              # 基础元素定义
│   ├── cell.go           # 单元格接口定义
│   ├── commonCell.go     # 通用单元格实现
│   ├── link.go           # 链路定义（连接多个单元格）
│   ├── trafficLightCell.go # 交通灯单元格实现
│   └── vehicle.go        # 车辆定义及行为逻辑
├── main.go               # 主程序入口
└── README.md             # 项目文档
```

## 核心组件

### 1. 单元格系统（Cell）

- **Cell接口**：定义了单元格的基本行为，如车辆的装载和卸载。
- **CommonCell**：基本单元格实现，支持车辆通行。
- **TrafficLightCell**：带交通灯控制的单元格，模拟十字路口等场景。

### 2. 链路系统（Link）

- 链路由多个单元格组成，表示道路。
- 提供车辆在单元格之间移动的通道。
- 支持快速获取链路上所有单元格的方法。

### 3. 车辆系统（Vehicle）

- 实现了纳格尔模型进行车辆行为模拟。
- 支持路径规划和导航。
- 包含加速、减速、随机慢行等车辆行为特性。

## 已完成的优化工作

### 1. CommonCell优化

- 将`sync.Mutex`替换为`sync.RWMutex`提高并发读取性能
- 预分配适当容量的map减少内存重分配
- 添加更详细的中文注释提高代码可读性
- 改进方法实现，减少不必要的锁竞争

### 2. TrafficLightCell优化

- 添加详细注释说明交通灯逻辑和相位控制
- 增强参数验证确保安全运行
- 优化相位切换逻辑

### 3. Link优化

- 实现原子计数器生成唯一单元格ID
- 改进并发安全处理
- 添加GetCell便捷方法
- 增强错误处理和日志记录

### 4. Vehicle优化

- 添加并发保护机制避免数据竞争
- 改进参数验证和错误处理
- 优化交叉路口通行逻辑
- 防止内存泄漏和数组越界

### 5. 轨迹记录系统优化

- 改进轨迹记录逻辑，由固定间隔改为按路径百分比设置检查点
- 根据路径长度动态调整记录点数量，短路径记录较少点，长路径记录较多点
- 确保始终记录起点和终点位置
- 大幅减少轨迹数据量，降低存储和分析成本
- 通过配置文件可自定义每辆车的记录点数量
- 移除周期性扫描机制，改为在车辆移动后立即检查检查点，提高系统性能
- 优化了内存使用，减少了不必要的数据结构复制
- 简化配置项，仅保留核心参数（enabled和traceRecordInterval），提高系统可维护性
- 固定使用合理的默认值替代可变配置参数，减少复杂性

## 使用方法

1. 创建交通网络：
```go
// 创建节点和链路
node1 := NewCommonCell(1, 5, 10) // ID, 最大速度, 容量
node2 := NewCommonCell(2, 5, 10)
link := NewLink(3, 500, 5, 20) // ID, 长度, 最大速度, 容量

// 连接节点
link.AddFromNode(node1)
link.AddToNode(node2)
```

2. 创建和配置车辆：
```go
vehicle := NewVehicle(1, 3, 1, 1.0, 0.3, false) // ID, 初始速度, 加速度, 占用空间, 随机减速概率, 是否固定车辆
vehicle.SetOD(graph, node1, node2)
vehicle.SetPath(path)
vehicle.BufferIn(0) // 进入时间为0
```

3. 运行模拟：
```go
for t := 0; t < simulationTime; t++ {
    // 更新交通灯
    trafficLight.Cycle()
    
    // 更新车辆
    vehicle.UpdateActiveState()
    if vehicle.State() == 3 && vehicle.IsActive() {
        vehicle.SystemIn()
    }
    if vehicle.State() == 4 {
        vehicle.Move(t)
    }
}
```

## 未来工作

- 添加更多交通场景模板
- 实现可视化界面
- 集成机器学习算法优化交通控制
- 添加并行处理提高大规模模拟性能

## 许可证

[添加许可证信息] 