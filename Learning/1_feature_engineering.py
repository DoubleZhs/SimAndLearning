from Method.FeatureEngineering import FeatureEngineering
from Method.CreateTimeFeature import CreateTimeFeature
import numpy as np


## 仿真设定
num_of_cells = 8000
gaps_between_traffic_light = 800
gaps_between_digs = 40
min_dist = 0
group_dist = 200

## 时序特征的生成设定
time_windows = 6
time_gap = 600
target = ['Travel Time']
time_feature_group = ['OD_Dig_2']
ignore_first_day = True


## 初始化两个类
fe = FeatureEngineering(num_of_cells,
                        gaps_between_traffic_light,
                        gaps_between_digs,
                        min_dist,
                        group_dist)

ctf = CreateTimeFeature(time_windows,
                        time_gap,
                        target,
                        time_feature_group,
                        ignore_first_day)


## 设定仿真数据的子文件路径 和 时间戳标识
sub_path = 1223
timestamp = 2024122316381124

# 因为是抽取，所以需要sampe_n
sample_n = 100

# 准备生成的比例列表
pro_Autonavi_lst = list(np.arange(0,0.51,0.05)[1:])
num_DiDi_lst = [10,20,30,40,50,60,70,80,90,100]




# 1、如果是常规生成特征，有多个n，每个n下再抽多个pro
# for n in num_DiDi_lst:
#     fe.run(sub_path,timestamp,n)
# ctf.create_time_feature(sub_path,num_DiDi_lst,pro_Autonavi_lst)



# 2、如果是在一个sample_n下，分别不断抽取n和pro
fe.run(sub_path,timestamp,sample_n)
ctf.create_time_feature_sample(sub_path,sample_n,num_DiDi_lst,pro_Autonavi_lst) 