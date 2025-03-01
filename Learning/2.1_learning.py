from Method.LearningForRing import LearningForRing
import numpy as np

# 初始设定
n_lst = [10,20,40,60,80,100]
pro_lst = [0.001,0.005,0.01] + list(np.arange(0,0.31,0.05)[1:])
date_lst = range(1,55)

# 特征设定
features = ['Origin','Destination','O Dig','D Dig',  # 起点与终点特征
            'Actual In Time','Hour','Quarter',   # 进入时间特征
            'Acceleration','SlowingPro',  # 车辆特征
            'PathLength','Traffic Light', # 路径特征            
            ] + [f'mean_TravelTime_before_{i}' for i in range(1,7)] + [f'std_TravelTime_before_{i}' for i in range(1,7)] # 时序特征
target = ['Travel Time']


# 初始化类
lfr = LearningForRing(n_lst,
                      pro_lst,
                      date_lst,
                      features,
                      target)

# 学习设定
sub_path = '1207'

k = 10  # 只在设定有天数上限时才有用
traindaynote = f'天数上限{k}'    # ['天数无上限','天数上限k']

testnote = '测试为下一天'   # ['测试周期最后','测试周期抽取','测试为下一天']

# 是否搜索参数
is_search = False

# 红绿灯变化的三个阶段日期
days_set = [(0,19),(20,34),(35,54)]


# 开始学习
lfr.learning(sub_path,
             traindaynote,
             testnote,
             is_search,
             k,
             days_set)