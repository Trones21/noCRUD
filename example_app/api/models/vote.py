from django.db import models
from rest_framework import serializers, viewsets
from .backstory_pitch import BackstoryPitch
from django.contrib.auth import get_user_model

User = get_user_model()

class Vote(models.Model):
    pitch = models.ForeignKey(BackstoryPitch, on_delete=models.CASCADE, related_name='votes')
    user = models.ForeignKey(User, on_delete=models.CASCADE)
    value = models.SmallIntegerField(choices=[(1, 'Upvote'), (-1, 'Downvote')])

    class Meta:
        unique_together = ('pitch', 'user')

class VoteSerializer(serializers.ModelSerializer):
    class Meta:
        model = Vote
        fields = '__all__'

class VoteViewSet(viewsets.ModelViewSet):
    queryset = Vote.objects.all()
    serializer_class = VoteSerializer