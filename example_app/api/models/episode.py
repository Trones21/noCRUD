from django.db import models
from rest_framework import serializers, viewsets
from .production import Production


class Episode(models.Model):
    production = models.ForeignKey(
        Production, on_delete=models.CASCADE, related_name="episodes"
    )
    season = models.IntegerField(blank=True, null=True)
    episode_number = models.IntegerField(blank=True, null=True)
    title = models.CharField(max_length=255)
    summary = models.TextField(blank=True)

    def __str__(self):
        return f"S{self.season:02}E{self.episode_number:02} - {self.title}"

    class Meta:
        """Settings for the model."""

        managed = True
        db_table = "episode"


class EpisodeSerializer(serializers.ModelSerializer):
    class Meta:
        model = Episode
        fields = "__all__"


class EpisodeViewSet(viewsets.ModelViewSet):
    queryset = Episode.objects.all()
    serializer_class = EpisodeSerializer
